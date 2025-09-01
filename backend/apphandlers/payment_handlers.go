package apphandlers

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/lightsparkdev/go-sdk/objects"
	uma_services "github.com/lightsparkdev/go-sdk/services"
	"github.com/lightsparkdev/go-sdk/webhooks"

	"tickets-by-uma/middleware"
	"tickets-by-uma/models"
	"tickets-by-uma/repositories"
	"tickets-by-uma/services"
)

type PaymentHandlers struct {
	paymentRepo repositories.PaymentRepository
	ticketRepo  repositories.TicketRepository
	umaService  services.UMAService
	client      *uma_services.LightsparkClient
	logger      *slog.Logger
}

func NewPaymentHandlers(
	paymentRepo repositories.PaymentRepository,
	ticketRepo repositories.TicketRepository,
	umaService services.UMAService,
	client *uma_services.LightsparkClient,
	logger *slog.Logger,
) *PaymentHandlers {
	return &PaymentHandlers{
		paymentRepo: paymentRepo,
		ticketRepo:  ticketRepo,
		umaService:  umaService,
		client:      client,
		logger:      logger,
	}
}

// HandlePaymentWebhook processes Lightning payment webhooks
func (h *PaymentHandlers) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	signingKey := os.Getenv("LIGHTSPARK_WEBHOOK_SIGNING_KEY")
	webhookData, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to decode webhook data", "error", err)
		middleware.WriteError(w, http.StatusBadRequest, "Invalid webhook data")
		return
	}

	h.logger.Info("Received payment webhook", "data", webhookData)

	event, err := webhooks.VerifyAndParse(webhookData, r.Header.Get(webhooks.SIGNATURE_HEADER), signingKey)
	if err != nil {
		log.Printf("Unable to verify and parse data: %v", err)
		http.Error(w, "Invalid webhook", http.StatusBadRequest)
		return
	}
	if event.EventType == objects.WebhookEventTypePaymentFinished {
		entityId := event.EntityId
		h.handlePaymentFinished(entityId)
	}

	// Return success response
	w.WriteHeader(http.StatusOK)

	//middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
	//	Message: "Payment webhook processed successfully",
	//	Data: map[string]interface{}{
	//		"payment_id": payment.ID,
	//		"status":     status,
	//	},
	//})
}

func (h *PaymentHandlers) handlePaymentFinished(entityID string) {
	h.logger.Info("Processing payment finished event", "entity_id", entityID)

	// Get the payment entity from Lightspark
	entity, err := h.client.GetEntity(entityID)
	if err != nil {
		h.logger.Error("Failed to fetch entity from Lightspark", "entity_id", entityID, "error", err)
		return
	}

	if entity == nil {
		h.logger.Error("Entity not found", "entity_id", entityID)
		return
	}

	// Cast entity to OutgoingPayment
	outgoingPayment, ok := (*entity).(objects.OutgoingPayment)
	if !ok {
		h.logger.Warn("Expected OutgoingPayment but got different type", "entity_id", entityID, "type", fmt.Sprintf("%T", *entity))
		return
	}

	h.logger.Info("Processing payment entity", "entity_id", entityID, "type", fmt.Sprintf("%T", *entity))

	// Get payment status
	paymentStatus := outgoingPayment.GetStatus()

	// Extract invoice ID from the OutgoingPayment - debug what we have
	h.logger.Info("Debugging payment request data", 
		"has_payment_request_data", outgoingPayment.PaymentRequestData != nil)
	
	var invoiceID string
	if outgoingPayment.PaymentRequestData != nil {
		// Log the actual type we're getting
		actualType := fmt.Sprintf("%T", *outgoingPayment.PaymentRequestData)
		h.logger.Info("PaymentRequestData type", "type", actualType)
		
		// Try different casting approaches
		switch paymentData := (*outgoingPayment.PaymentRequestData).(type) {
		case *objects.InvoiceData:
			invoiceID = paymentData.GetEncodedPaymentRequest()
			if len(invoiceID) > 50 {
				h.logger.Info("Got invoice ID from InvoiceData", "invoice_id", invoiceID[:50]+"...")
			} else {
				h.logger.Info("Got invoice ID from InvoiceData", "invoice_id", invoiceID)
			}
		case objects.InvoiceData:
			invoiceID = paymentData.GetEncodedPaymentRequest()
			if len(invoiceID) > 50 {
				h.logger.Info("Got invoice ID from InvoiceData (value type)", "invoice_id", invoiceID[:50]+"...")
			} else {
				h.logger.Info("Got invoice ID from InvoiceData (value type)", "invoice_id", invoiceID)
			}
		default:
			h.logger.Warn("PaymentRequestData is not InvoiceData", "actual_type", actualType)
		}
	} else {
		h.logger.Warn("PaymentRequestData is nil")
	}
	
	if invoiceID == "" {
		h.logger.Error("Could not extract invoice ID from OutgoingPayment", "entity_id", entityID)
		return
	}

	h.logger.Info("Processing outgoing payment",
		"invoice_id", invoiceID,
		"status", paymentStatus,
		"amount", outgoingPayment.GetAmount())

	// Get payment record by invoice ID
	payment, err := h.paymentRepo.GetByInvoiceID(invoiceID)
	if err != nil {
		h.logger.Error("Failed to fetch payment", "invoice_id", invoiceID, "error", err)
		return
	}

	if payment == nil {
		h.logger.Error("Payment not found", "invoice_id", invoiceID)
		return
	}

	status := "paid" // Payment finished means it was successful

	// Update payment status
	oldStatus := payment.Status
	if err := h.paymentRepo.UpdateStatus(payment.ID, status); err != nil {
		h.logger.Error("Failed to update payment status", "payment_id", payment.ID, "error", err)
		return
	}

	// Update ticket payment status
	if err := h.ticketRepo.UpdatePaymentStatus(payment.TicketID, status); err != nil {
		h.logger.Error("Failed to update ticket payment status", "ticket_id", payment.TicketID, "error", err)
		return
	}

	// Process UMA callback
	if err := h.umaService.HandleUMACallback(invoiceID, status); err != nil {
		h.logger.Error("Failed to process UMA callback", "invoice_id", invoiceID, "error", err)
		// Don't fail the webhook for UMA callback errors
	}

	h.logger.Info("Payment finished processed successfully",
		"payment_id", payment.ID,
		"invoice_id", invoiceID,
		"old_status", oldStatus,
		"new_status", status)
}

// HandlePaymentStatus checks the status of a payment by invoice ID
func (h *PaymentHandlers) HandlePaymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invoiceID := vars["invoice_id"]

	if invoiceID == "" {
		middleware.WriteError(w, http.StatusBadRequest, "Invoice ID is required")
		return
	}

	h.logger.Info("Checking payment status", "invoice_id", invoiceID)

	// Get payment record
	payment, err := h.paymentRepo.GetByInvoiceID(invoiceID)
	if err != nil {
		h.logger.Error("Failed to fetch payment", "invoice_id", invoiceID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch payment")
		return
	}

	if payment == nil {
		middleware.WriteError(w, http.StatusNotFound, "Payment not found")
		return
	}

	// Get ticket information
	ticket, err := h.ticketRepo.GetByID(payment.TicketID)
	if err != nil {
		h.logger.Error("Failed to fetch ticket", "ticket_id", payment.TicketID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch ticket")
		return
	}

	// Get payment status from UMA service
	umaStatus, err := h.umaService.CheckPaymentStatus(invoiceID)
	if err != nil {
		h.logger.Warn("Failed to get UMA payment status", "invoice_id", invoiceID, "error", err)
		// Continue with database status if UMA service fails
	}

	// Combine database and UMA status
	statusResponse := map[string]interface{}{
		"payment": map[string]interface{}{
			"id":          payment.ID,
			"invoice_id":  payment.InvoiceID,
			"amount_sats": payment.Amount,
			"status":      payment.Status,
			"paid_at":     payment.PaidAt,
			"created_at":  payment.CreatedAt,
		},
		"ticket": map[string]interface{}{
			"id":             ticket.ID,
			"ticket_code":    ticket.TicketCode,
			"payment_status": ticket.PaymentStatus,
		},
	}

	// Add UMA status if available
	if umaStatus != nil {
		statusResponse["uma_status"] = umaStatus
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Payment status retrieved successfully",
		Data:    statusResponse,
	})
}

// HandleGetPendingPayments gets all pending payments (admin only)
func (h *PaymentHandlers) HandleGetPendingPayments(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Fetching pending payments")

	payments, err := h.paymentRepo.GetPendingPayments()
	if err != nil {
		h.logger.Error("Failed to fetch pending payments", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch pending payments")
		return
	}

	// Enrich payments with ticket and event information
	paymentDetails := make([]map[string]interface{}, 0, len(payments))
	for _, payment := range payments {
		ticket, err := h.ticketRepo.GetByID(payment.TicketID)
		if err != nil {
			h.logger.Warn("Failed to fetch ticket for payment", "payment_id", payment.ID, "ticket_id", payment.TicketID, "error", err)
			continue
		}

		paymentDetail := map[string]interface{}{
			"id":          payment.ID,
			"invoice_id":  payment.InvoiceID,
			"amount_sats": payment.Amount,
			"status":      payment.Status,
			"created_at":  payment.CreatedAt,
			"ticket": map[string]interface{}{
				"id":             ticket.ID,
				"ticket_code":    ticket.TicketCode,
				"payment_status": ticket.PaymentStatus,
				"uma_address":    ticket.UMAAddress,
			},
		}
		paymentDetails = append(paymentDetails, paymentDetail)
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Pending payments retrieved successfully",
		Data:    paymentDetails,
	})
}

// HandleRetryPayment retries a failed payment (admin only)
func (h *PaymentHandlers) HandleRetryPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	paymentID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid payment ID")
		return
	}

	h.logger.Info("Retrying payment", "payment_id", paymentID)

	// Get payment record
	payment, err := h.paymentRepo.GetByID(paymentID)
	if err != nil {
		h.logger.Error("Failed to fetch payment", "payment_id", paymentID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch payment")
		return
	}

	if payment == nil {
		middleware.WriteError(w, http.StatusNotFound, "Payment not found")
		return
	}

	// Check if payment can be retried
	if payment.Status != "failed" && payment.Status != "expired" {
		middleware.WriteError(w, http.StatusBadRequest, "Payment cannot be retried")
		return
	}

	// Get ticket information
	ticket, err := h.ticketRepo.GetByID(payment.TicketID)
	if err != nil {
		h.logger.Error("Failed to fetch ticket", "ticket_id", payment.TicketID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch ticket")
		return
	}

	// Create new UMA Request for retry payment
	invoice, err := h.umaService.CreateUMARequest(
		ticket.UMAAddress,
		payment.Amount,
		fmt.Sprintf("Retry payment for ticket %s", ticket.TicketCode),
		true, // isAdmin = true for admin endpoints
	)
	if err != nil {
		h.logger.Error("Failed to create retry invoice", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create retry invoice")
		return
	}

	// Update payment with new invoice
	payment.InvoiceID = invoice.ID
	payment.Status = "pending"
	payment.PaidAt = nil

	if err := h.paymentRepo.Update(payment); err != nil {
		h.logger.Error("Failed to update payment for retry", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to update payment")
		return
	}

	// Update ticket status
	if err := h.ticketRepo.UpdatePaymentStatus(ticket.ID, "pending"); err != nil {
		h.logger.Error("Failed to update ticket status for retry", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to update ticket status")
		return
	}

	h.logger.Info("Payment retry initiated",
		"payment_id", paymentID,
		"new_invoice_id", invoice.ID)

	retryResponse := map[string]interface{}{
		"payment_id": paymentID,
		"new_invoice": map[string]interface{}{
			"id":          invoice.ID,
			"bolt11":      invoice.Bolt11,
			"amount_sats": invoice.AmountSats,
			"expires_at":  invoice.ExpiresAt,
		},
		"status": "pending",
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Payment retry initiated successfully",
		Data:    retryResponse,
	})
}
