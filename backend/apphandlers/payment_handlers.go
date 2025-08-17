package apphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"log/slog"
	"tickets-by-uma/middleware"
	"tickets-by-uma/models"
	"tickets-by-uma/repositories"
	"tickets-by-uma/services"
)

type PaymentHandlers struct {
	paymentRepo repositories.PaymentRepository
	ticketRepo  repositories.TicketRepository
	umaService  services.UMAService
	logger      *slog.Logger
}

func NewPaymentHandlers(
	paymentRepo repositories.PaymentRepository,
	ticketRepo repositories.TicketRepository,
	umaService services.UMAService,
	logger *slog.Logger,
) *PaymentHandlers {
	return &PaymentHandlers{
		paymentRepo: paymentRepo,
		ticketRepo:  ticketRepo,
		umaService:  umaService,
		logger:      logger,
	}
}

// HandlePaymentWebhook processes Lightning payment webhooks
func (h *PaymentHandlers) HandlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	var webhookData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&webhookData); err != nil {
		h.logger.Error("Failed to decode webhook data", "error", err)
		middleware.WriteError(w, http.StatusBadRequest, "Invalid webhook data")
		return
	}
	
	h.logger.Info("Received payment webhook", "data", webhookData)
	
	// Extract payment information from webhook
	invoiceID, ok := webhookData["invoice_id"].(string)
	if !ok {
		h.logger.Error("Missing invoice_id in webhook")
		middleware.WriteError(w, http.StatusBadRequest, "Missing invoice_id")
		return
	}
	
	status, ok := webhookData["status"].(string)
	if !ok {
		h.logger.Error("Missing status in webhook")
		middleware.WriteError(w, http.StatusBadRequest, "Missing status")
		return
	}
	
	paymentHash, _ := webhookData["payment_hash"].(string)
	
	h.logger.Info("Processing payment webhook", 
		"invoice_id", invoiceID, 
		"status", status,
		"payment_hash", paymentHash)
	
	// Get payment record by invoice ID
	payment, err := h.paymentRepo.GetByInvoiceID(invoiceID)
	if err != nil {
		h.logger.Error("Failed to fetch payment", "invoice_id", invoiceID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch payment")
		return
	}
	
	if payment == nil {
		h.logger.Error("Payment not found", "invoice_id", invoiceID)
		middleware.WriteError(w, http.StatusNotFound, "Payment not found")
		return
	}
	
	// Update payment status
	oldStatus := payment.Status
	if err := h.paymentRepo.UpdateStatus(payment.ID, status); err != nil {
		h.logger.Error("Failed to update payment status", "payment_id", payment.ID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to update payment status")
		return
	}
	
	// Update ticket payment status
	if err := h.ticketRepo.UpdatePaymentStatus(payment.TicketID, status); err != nil {
		h.logger.Error("Failed to update ticket payment status", "ticket_id", payment.TicketID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to update ticket payment status")
		return
	}
	
	// Process UMA callback
	if err := h.umaService.HandleUMACallback(paymentHash, status); err != nil {
		h.logger.Error("Failed to process UMA callback", "error", err)
		// Don't fail the webhook for UMA callback errors
	}
	
	h.logger.Info("Payment webhook processed successfully", 
		"payment_id", payment.ID,
		"old_status", oldStatus,
		"new_status", status)
	
	// Return success response
	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Payment webhook processed successfully",
		Data: map[string]interface{}{
			"payment_id": payment.ID,
			"status":     status,
		},
	})
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
	
	// Create new UMA invoice
	invoice, err := h.umaService.CreateInvoice(
		ticket.UMAAddress, 
		payment.Amount, 
		fmt.Sprintf("Retry payment for ticket %s", ticket.TicketCode),
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
