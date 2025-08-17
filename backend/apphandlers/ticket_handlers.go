package apphandlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"log/slog"
	"tickets-by-uma/middleware"
	"tickets-by-uma/models"
	"tickets-by-uma/repositories"
	"tickets-by-uma/services"
)

type TicketHandlers struct {
	ticketRepo  repositories.TicketRepository
	eventRepo   repositories.EventRepository
	paymentRepo repositories.PaymentRepository
	umaService  services.UMAService
	logger      *slog.Logger
}

func NewTicketHandlers(
	ticketRepo repositories.TicketRepository,
	eventRepo repositories.EventRepository,
	paymentRepo repositories.PaymentRepository,
	umaService services.UMAService,
	logger *slog.Logger,
) *TicketHandlers {
	return &TicketHandlers{
		ticketRepo:  ticketRepo,
		eventRepo:   eventRepo,
		paymentRepo: paymentRepo,
		umaService:  umaService,
		logger:      logger,
	}
}

// HandlePurchaseTicket initiates a ticket purchase with UMA payment
func (h *TicketHandlers) HandlePurchaseTicket(w http.ResponseWriter, r *http.Request) {
	var req models.TicketPurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Validate request
	if err := h.validatePurchaseRequest(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	h.logger.Info("Processing ticket purchase", 
		"event_id", req.EventID, 
		"user_id", req.UserID,
		"uma_address", req.UMAAddress)
	
	// Check if event exists and is active
	event, err := h.eventRepo.GetByID(req.EventID)
	if err != nil {
		h.logger.Error("Failed to fetch event", "event_id", req.EventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}
	
	if event == nil {
		middleware.WriteError(w, http.StatusNotFound, "Event not found")
		return
	}
	
	if !event.IsActive {
		middleware.WriteError(w, http.StatusBadRequest, "Event is not active")
		return
	}
	
	// Check if event has available capacity
	availableTickets, err := h.eventRepo.GetAvailableTicketCount(req.EventID)
	if err != nil {
		h.logger.Error("Failed to check event capacity", "event_id", req.EventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to check event capacity")
		return
	}
	
	if availableTickets <= 0 {
		middleware.WriteError(w, http.StatusBadRequest, "Event is sold out")
		return
	}
	
	// Validate UMA address
	if err := h.umaService.ValidateUMAAddress(req.UMAAddress); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid UMA address: %v", err))
		return
	}
	
	// Create UMA invoice
	invoice, err := h.umaService.CreateInvoice(req.UMAAddress, event.PriceSats, fmt.Sprintf("Ticket for %s", event.Title))
	if err != nil {
		h.logger.Error("Failed to create UMA invoice", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create payment invoice")
		return
	}
	
	// Generate unique ticket code
	ticketCode, err := middleware.GenerateTicketCode()
	if err != nil {
		h.logger.Error("Failed to generate ticket code", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to generate ticket code")
		return
	}
	
	// Create ticket record
	ticket := &models.Ticket{
		EventID:       req.EventID,
		UserID:        req.UserID,
		TicketCode:    ticketCode,
		PaymentStatus: "pending",
		InvoiceID:     invoice.ID,
		UMAAddress:    req.UMAAddress,
	}
	
	if err := h.ticketRepo.Create(ticket); err != nil {
		h.logger.Error("Failed to create ticket", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create ticket")
		return
	}
	
	// Create payment record
	payment := &models.Payment{
		TicketID:  ticket.ID,
		InvoiceID: invoice.ID,
		Amount:    event.PriceSats,
		Status:    "pending",
	}
	
	if err := h.paymentRepo.Create(payment); err != nil {
		h.logger.Error("Failed to create payment record", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create payment record")
		return
	}
	
	h.logger.Info("Ticket purchase initiated", 
		"ticket_id", ticket.ID,
		"invoice_id", invoice.ID)
	
	// Return ticket and invoice information
	response := map[string]interface{}{
		"ticket": map[string]interface{}{
			"id":             ticket.ID,
			"ticket_code":    ticket.TicketCode,
			"payment_status": ticket.PaymentStatus,
		},
		"invoice": map[string]interface{}{
			"id":          invoice.ID,
			"bolt11":      invoice.Bolt11,
			"amount_sats": invoice.AmountSats,
			"expires_at":  invoice.ExpiresAt,
		},
		"event": map[string]interface{}{
			"title": event.Title,
			"price_sats": event.PriceSats,
		},
	}
	
	middleware.WriteJSON(w, http.StatusCreated, models.SuccessResponse{
		Message: "Ticket purchase initiated successfully",
		Data:    response,
	})
}

// HandleTicketStatus checks the payment status of a ticket
func (h *TicketHandlers) HandleTicketStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid ticket ID")
		return
	}
	
	h.logger.Info("Checking ticket status", "ticket_id", ticketID)
	
	ticket, err := h.ticketRepo.GetByID(ticketID)
	if err != nil {
		h.logger.Error("Failed to fetch ticket", "ticket_id", ticketID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch ticket")
		return
	}
	
	if ticket == nil {
		middleware.WriteError(w, http.StatusNotFound, "Ticket not found")
		return
	}
	
	// Get payment information
	payment, err := h.paymentRepo.GetByTicketID(ticketID)
	if err != nil {
		h.logger.Error("Failed to fetch payment", "ticket_id", ticketID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch payment")
		return
	}
	
	// Get event information
	event, err := h.eventRepo.GetByID(ticket.EventID)
	if err != nil {
		h.logger.Error("Failed to fetch event", "event_id", ticket.EventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}
	
	statusResponse := map[string]interface{}{
		"ticket": map[string]interface{}{
			"id":             ticket.ID,
			"ticket_code":    ticket.TicketCode,
			"payment_status": ticket.PaymentStatus,
			"created_at":     ticket.CreatedAt,
			"paid_at":        ticket.PaidAt,
		},
		"payment": map[string]interface{}{
			"status":      payment.Status,
			"amount_sats": payment.Amount,
			"invoice_id":  payment.InvoiceID,
		},
		"event": map[string]interface{}{
			"title":     event.Title,
			"start_time": event.StartTime,
		},
	}
	
	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Ticket status retrieved successfully",
		Data:    statusResponse,
	})
}

// HandleValidateTicket validates a ticket for event access
func (h *TicketHandlers) HandleValidateTicket(w http.ResponseWriter, r *http.Request) {
	var req models.TicketValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if req.TicketCode == "" {
		middleware.WriteError(w, http.StatusBadRequest, "Ticket code is required")
		return
	}
	
	if req.EventID <= 0 {
		middleware.WriteError(w, http.StatusBadRequest, "Valid event ID is required")
		return
	}
	
	h.logger.Info("Validating ticket", "ticket_code", req.TicketCode, "event_id", req.EventID)
	
	// Get ticket by code
	ticket, err := h.ticketRepo.GetByTicketCode(req.TicketCode)
	if err != nil {
		h.logger.Error("Failed to fetch ticket", "ticket_code", req.TicketCode, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch ticket")
		return
	}
	
	if ticket == nil {
		middleware.WriteError(w, http.StatusNotFound, "Ticket not found")
		return
	}
	
	// Check if ticket is for the correct event
	if ticket.EventID != req.EventID {
		middleware.WriteError(w, http.StatusBadRequest, "Ticket is not valid for this event")
		return
	}
	
	// Check if ticket is paid
	if ticket.PaymentStatus != "paid" {
		middleware.WriteError(w, http.StatusBadRequest, "Ticket payment is not complete")
		return
	}
	
	// Get event information
	event, err := h.eventRepo.GetByID(req.EventID)
	if err != nil {
		h.logger.Error("Failed to fetch event", "event_id", req.EventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}
	
	if event == nil {
		middleware.WriteError(w, http.StatusNotFound, "Event not found")
		return
	}
	
	// Check if event is currently active
	now := time.Now()
	if now.Before(event.StartTime) || now.After(event.EndTime) {
		middleware.WriteError(w, http.StatusBadRequest, "Event is not currently active")
		return
	}
	
	h.logger.Info("Ticket validated successfully", "ticket_code", req.TicketCode)
	
	validationResponse := map[string]interface{}{
		"valid": true,
		"ticket": map[string]interface{}{
			"id":          ticket.ID,
			"ticket_code": ticket.TicketCode,
			"user_id":     ticket.UserID,
		},
		"event": map[string]interface{}{
			"title":     event.Title,
			"stream_url": event.StreamURL,
		},
		"validated_at": time.Now(),
	}
	
	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Ticket validated successfully",
		Data:    validationResponse,
	})
}

// HandleGetUserTickets gets all tickets for a specific user
func (h *TicketHandlers) HandleGetUserTickets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["user_id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}
	
	h.logger.Info("Fetching user tickets", "user_id", userID)
	
	tickets, err := h.ticketRepo.GetByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to fetch user tickets", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch user tickets")
		return
	}
	
	// Enrich tickets with event information
	ticketDetails := make([]map[string]interface{}, 0, len(tickets))
	for _, ticket := range tickets {
		event, err := h.eventRepo.GetByID(ticket.EventID)
		if err != nil {
			h.logger.Warn("Failed to fetch event for ticket", "ticket_id", ticket.ID, "event_id", ticket.EventID, "error", err)
			continue
		}
		
		ticketDetail := map[string]interface{}{
			"id":             ticket.ID,
			"ticket_code":    ticket.TicketCode,
			"payment_status": ticket.PaymentStatus,
			"created_at":     ticket.CreatedAt,
			"paid_at":        ticket.PaidAt,
			"event": map[string]interface{}{
				"id":          event.ID,
				"title":       event.Title,
				"start_time":  event.StartTime,
				"stream_url":  event.StreamURL,
			},
		}
		ticketDetails = append(ticketDetails, ticketDetail)
	}
	
	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "User tickets retrieved successfully",
		Data:    ticketDetails,
	})
}

// validatePurchaseRequest validates the ticket purchase request
func (h *TicketHandlers) validatePurchaseRequest(req *models.TicketPurchaseRequest) error {
	if req.EventID <= 0 {
		return fmt.Errorf("valid event ID is required")
	}
	
	if req.UserID <= 0 {
		return fmt.Errorf("valid user ID is required")
	}
	
	if req.UMAAddress == "" {
		return fmt.Errorf("UMA address is required")
	}
	
	return nil
}
