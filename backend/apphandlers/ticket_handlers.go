package apphandlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

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
	event, err := h.eventRepo.GetByIDWithUMAInvoice(req.EventID)
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

	// Check if event has a UMA Request invoice (business-created)
	// Free events (price = 0) don't need UMA invoices since tickets are free
	// Paid events (price > 0) require UMA invoices for payment processing
	if event.PriceSats > 0 && event.UMARequestInvoice == nil {
		h.logger.Error("Paid event missing UMA Request invoice",
			"event_id", req.EventID,
			"price_sats", event.PriceSats)
		middleware.WriteError(w, http.StatusServiceUnavailable, "Event payment system not configured")
		return
	}

	if event.PriceSats == 0 {
		h.logger.Info("Free event - no payment required",
			"event_id", req.EventID,
			"price_sats", event.PriceSats)
		// For free events, we can proceed without UMA invoice
		// The ticket will be created with payment_status = 'free'
	} else if event.UMARequestInvoice == nil {
		h.logger.Error("Paid event missing UMA Request invoice",
			"event_id", req.EventID,
			"price_sats", event.PriceSats)
		middleware.WriteError(w, http.StatusServiceUnavailable, "Event payment system not configured")
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

	// Generate unique ticket code
	ticketCode, err := middleware.GenerateTicketCode()
	if err != nil {
		h.logger.Error("Failed to generate ticket code", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to generate ticket code")
		return
	}

	var ticket *models.Ticket
	var payment *models.Payment

	if event.PriceSats == 0 {
		// Free event - create ticket with 'free' payment status
		h.logger.Info("Creating free ticket for free event",
			"event_id", req.EventID,
			"price_sats", event.PriceSats)

		ticket = &models.Ticket{
			EventID:       req.EventID,
			UserID:        req.UserID,
			TicketCode:    ticketCode,
			PaymentStatus: "free", // Free tickets don't need payment
			InvoiceID:     "",     // No invoice for free tickets
			UMAAddress:    req.UMAAddress,
		}

		if err := h.ticketRepo.Create(ticket); err != nil {
			h.logger.Error("Failed to create free ticket", "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create ticket")
			return
		}

		// No payment record needed for free tickets
		payment = nil

	} else {
		// Paid event - use pre-created UMA Request invoice
		if err := h.umaService.ValidateUMAAddress(req.UMAAddress); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid UMA address: %v", err))
			return
		}

		// Check if event has UMA Request invoice
		if event.UMARequestInvoice == nil {
			h.logger.Error("Paid event missing UMA Request invoice",
				"event_id", req.EventID,
				"price_sats", event.PriceSats)
			middleware.WriteError(w, http.StatusServiceUnavailable, "Event payment system not configured - admin must create UMA Request invoice")
			return
		}

		h.logger.Info("Using UMA Request invoice for ticket purchase",
			"uma_address", req.UMAAddress,
			"amount_sats", event.PriceSats,
			"event_id", req.EventID,
			"uma_invoice_id", event.UMARequestInvoice.InvoiceID)

		// Create ticket record with pending payment
		ticket = &models.Ticket{
			EventID:       req.EventID,
			UserID:        req.UserID,
			TicketCode:    ticketCode,
			PaymentStatus: "pending",
			InvoiceID:     event.UMARequestInvoice.InvoiceID,
			UMAAddress:    req.UMAAddress,
		}

		if err := h.ticketRepo.Create(ticket); err != nil {
			h.logger.Error("Failed to create paid ticket", "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create ticket")
			return
		}

		// Create payment record using UMA Request invoice
		payment = &models.Payment{
			TicketID:  ticket.ID,
			InvoiceID: event.UMARequestInvoice.InvoiceID,
			Amount:    event.UMARequestInvoice.AmountSats,
			Status:    "pending",
		}

		if err := h.paymentRepo.Create(payment); err != nil {
			h.logger.Error("Failed to create payment record", "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create payment record")
			return
		}

		h.logger.Info("Ticket created with UMA Request payment",
			"ticket_id", ticket.ID,
			"uma_invoice_id", event.UMARequestInvoice.InvoiceID,
			"uma_address", req.UMAAddress)
	}

	// Return ticket and event information
	response := map[string]interface{}{
		"ticket": map[string]interface{}{
			"id":             ticket.ID,
			"ticket_code":    ticket.TicketCode,
			"payment_status": ticket.PaymentStatus,
		},
		"event": map[string]interface{}{
			"title":      event.Title,
			"price_sats": event.PriceSats,
		},
	}

	// Add UMA Request invoice information for paid events
	if event.PriceSats > 0 && event.UMARequestInvoice != nil {
		response["uma_request"] = map[string]interface{}{
			"invoice_id":   event.UMARequestInvoice.InvoiceID,
			"bolt11":       event.UMARequestInvoice.Bolt11, // UMA Request uses Lightning under the hood
			"amount_sats":  event.UMARequestInvoice.AmountSats,
			"payment_hash": event.UMARequestInvoice.PaymentHash,
			"uma_address":  event.UMARequestInvoice.UMAAddress,
			"description":  event.UMARequestInvoice.Description,
			"expires_at":   event.UMARequestInvoice.ExpiresAt,
			"status":       "pending",
		}
		response["payment_required"] = true
	}

	var message string
	if event.PriceSats == 0 {
		message = "Free ticket created successfully"
	} else {
		message = "Ticket purchase initiated successfully using UMA Request invoice"
	}

	middleware.WriteJSON(w, http.StatusCreated, models.SuccessResponse{
		Message: message,
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
			"title":      event.Title,
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
			"title":      event.Title,
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

	h.logger.Info("Fetching tickets for user", "user_id", userID)

	tickets, err := h.ticketRepo.GetByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to fetch user tickets", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch tickets")
		return
	}

	// Enrich tickets with event information
	enrichedTickets := make([]map[string]interface{}, 0, len(tickets))
	for _, ticket := range tickets {
		// Get event information for each ticket
		event, err := h.eventRepo.GetByID(ticket.EventID)
		if err != nil {
			h.logger.Error("Failed to fetch event for ticket", "ticket_id", ticket.ID, "event_id", ticket.EventID, "error", err)
			continue
		}

		// Get payment information for each ticket
		payment, err := h.paymentRepo.GetByTicketID(ticket.ID)
		if err != nil {
			h.logger.Error("Failed to fetch payment for ticket", "ticket_id", ticket.ID, "error", err)
		}

		enrichedTicket := map[string]interface{}{
			"id":             ticket.ID,
			"ticket_code":    ticket.TicketCode,
			"payment_status": ticket.PaymentStatus,
			"uma_address":    ticket.UMAAddress,
			"created_at":     ticket.CreatedAt,
			"updated_at":     ticket.UpdatedAt,
			"event": map[string]interface{}{
				"id":         event.ID,
				"title":      event.Title,
				"start_time": event.StartTime,
				"end_time":   event.EndTime,
				"stream_url": event.StreamURL,
				"price_sats": event.PriceSats,
			},
		}

		// Add payment information if available
		if payment != nil {
			enrichedTicket["payment"] = map[string]interface{}{
				"id":     payment.ID,
				"status": payment.Status,
				"amount": payment.Amount,
			}
		}

		enrichedTickets = append(enrichedTickets, enrichedTicket)
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "User tickets retrieved successfully",
		Data:    enrichedTickets,
	})
}

// HandleUMAPaymentCallback processes UMA payment callbacks
func (h *TicketHandlers) HandleUMAPaymentCallback(w http.ResponseWriter, r *http.Request) {
	var req models.UMACallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid callback request body")
		return
	}

	h.logger.Info("Processing UMA payment callback",
		"payment_hash", req.PaymentHash,
		"status", req.Status,
		"invoice_id", req.InvoiceID)

	// Process the UMA callback
	if err := h.umaService.HandleUMACallback(req.PaymentHash, req.Status); err != nil {
		h.logger.Error("Failed to process UMA callback", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to process payment callback")
		return
	}

	// Update ticket status based on payment status
	if req.Status == "paid" {
		// Find ticket by invoice ID and update status
		ticket, err := h.ticketRepo.GetByInvoiceID(req.InvoiceID)
		if err != nil {
			h.logger.Error("Failed to find ticket for invoice", "invoice_id", req.InvoiceID, "error", err)
		} else if ticket != nil {
			// Update ticket status to paid
			ticket.PaymentStatus = "paid"
			now := time.Now()
			ticket.PaidAt = &now

			if err := h.ticketRepo.Update(ticket); err != nil {
				h.logger.Error("Failed to update ticket status", "ticket_id", ticket.ID, "error", err)
			}

			// Update payment record
			payment, err := h.paymentRepo.GetByInvoiceID(req.InvoiceID)
			if err == nil && payment != nil {
				payment.Status = "paid"
				payment.PaidAt = &now
				h.paymentRepo.Update(payment)
			}
		}
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Payment callback processed successfully",
		Data:    map[string]string{"status": "processed"},
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
