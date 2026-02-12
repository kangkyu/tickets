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

	h.logger.Info("Processing payment entity", "entity_id", entityID, "type", fmt.Sprintf("%T", *entity))

	// Try IncomingPayment first (when someone pays our invoice from their wallet)
	if incomingPayment, ok := (*entity).(objects.IncomingPayment); ok {
		h.handleIncomingPayment(entityID, incomingPayment)
		return
	}

	// Try OutgoingPayment (for backwards compatibility / self-pay scenarios)
	if outgoingPayment, ok := (*entity).(objects.OutgoingPayment); ok {
		h.handleOutgoingPayment(entityID, outgoingPayment)
		return
	}

	h.logger.Warn("Entity is neither IncomingPayment nor OutgoingPayment",
		"entity_id", entityID, "type", fmt.Sprintf("%T", *entity))
}

// handleIncomingPayment processes a payment received on our node (someone paid our invoice).
func (h *PaymentHandlers) handleIncomingPayment(entityID string, incomingPayment objects.IncomingPayment) {
	h.logger.Info("Processing incoming payment",
		"entity_id", entityID,
		"amount", incomingPayment.Amount,
		"is_uma", incomingPayment.IsUma)

	// Get the PaymentRequest (Invoice) reference from the IncomingPayment
	if incomingPayment.PaymentRequest == nil {
		h.logger.Error("IncomingPayment has no PaymentRequest (keysend payment?)", "entity_id", entityID)
		return
	}

	invoiceEntityID := incomingPayment.PaymentRequest.Id
	h.logger.Info("Fetching invoice for incoming payment", "invoice_entity_id", invoiceEntityID)

	// Fetch the full Invoice entity to get the bolt11
	invoiceEntity, err := h.client.GetEntity(invoiceEntityID)
	if err != nil {
		h.logger.Error("Failed to fetch invoice entity", "invoice_id", invoiceEntityID, "error", err)
		return
	}

	if invoiceEntity == nil {
		h.logger.Error("Invoice entity not found", "invoice_id", invoiceEntityID)
		return
	}

	invoice, ok := (*invoiceEntity).(objects.Invoice)
	if !ok {
		h.logger.Error("Entity is not an Invoice", "invoice_id", invoiceEntityID, "type", fmt.Sprintf("%T", *invoiceEntity))
		return
	}

	bolt11 := invoice.Data.EncodedPaymentRequest
	h.logger.Info("Extracted bolt11 from invoice",
		"invoice_id", invoiceEntityID,
		"bolt11_prefix", bolt11[:min(len(bolt11), 50)]+"...")

	// Match the bolt11 to our payment record in the database
	h.markPaymentPaid(bolt11)
}

// handleOutgoingPayment processes an outgoing payment (backwards compatibility).
func (h *PaymentHandlers) handleOutgoingPayment(entityID string, outgoingPayment objects.OutgoingPayment) {
	h.logger.Info("Processing outgoing payment", "entity_id", entityID)

	// Extract bolt11 from the OutgoingPayment
	var bolt11 string
	if outgoingPayment.PaymentRequestData != nil {
		if encodedReq, ok := (*outgoingPayment.PaymentRequestData).(objects.InvoiceData); ok {
			bolt11 = encodedReq.GetEncodedPaymentRequest()
		} else {
			h.logger.Warn("PaymentRequestData doesn't implement InvoiceData")
		}
	} else {
		h.logger.Warn("PaymentRequestData is nil")
	}

	if bolt11 == "" {
		h.logger.Error("Could not extract bolt11 from OutgoingPayment", "entity_id", entityID)
		return
	}

	h.logger.Info("Processing outgoing payment",
		"bolt11_prefix", bolt11[:min(len(bolt11), 50)]+"...",
		"status", outgoingPayment.GetStatus(),
		"amount", outgoingPayment.GetAmount())

	h.markPaymentPaid(bolt11)
}

// markPaymentPaid looks up a payment by bolt11 and marks it and its ticket as paid.
func (h *PaymentHandlers) markPaymentPaid(bolt11 string) {
	payment, err := h.paymentRepo.GetByInvoiceID(bolt11)
	if err != nil {
		h.logger.Error("Failed to fetch payment by bolt11", "error", err)
		return
	}

	if payment == nil {
		h.logger.Error("Payment not found in database for bolt11",
			"bolt11_prefix", bolt11[:min(len(bolt11), 50)]+"...")
		return
	}

	status := "paid"
	oldStatus := payment.Status

	// Update payment status
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
	if err := h.umaService.HandleUMACallback(bolt11, status); err != nil {
		h.logger.Error("Failed to process UMA callback", "error", err)
	}

	h.logger.Info("Payment marked as paid successfully",
		"payment_id", payment.ID,
		"ticket_id", payment.TicketID,
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

