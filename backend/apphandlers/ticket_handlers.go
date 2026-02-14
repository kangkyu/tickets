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
	umaRepo     repositories.UMARequestInvoiceRepository
	nwcRepo     repositories.NWCConnectionRepository
	umaService  services.UMAService
	logger      *slog.Logger
	domain      string
}

func NewTicketHandlers(
	ticketRepo repositories.TicketRepository,
	eventRepo repositories.EventRepository,
	paymentRepo repositories.PaymentRepository,
	umaRepo repositories.UMARequestInvoiceRepository,
	nwcRepo repositories.NWCConnectionRepository,
	umaService services.UMAService,
	logger *slog.Logger,
	domain string,
) *TicketHandlers {
	return &TicketHandlers{
		ticketRepo:  ticketRepo,
		eventRepo:   eventRepo,
		paymentRepo: paymentRepo,
		umaRepo:     umaRepo,
		nwcRepo:     nwcRepo,
		umaService:  umaService,
		logger:      logger,
		domain:      domain,
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

	// Generate unique ticket code
	ticketCode, err := middleware.GenerateTicketCode()
	if err != nil {
		h.logger.Error("Failed to generate ticket code", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to generate ticket code")
		return
	}

	var ticket *models.Ticket
	var ticketInvoice *models.UMARequestInvoice

	if event.PriceSats == 0 {
		// Free event - create ticket with 'paid' payment status (since it's free)
		h.logger.Info("Creating free ticket for free event",
			"event_id", req.EventID,
			"price_sats", event.PriceSats)

		ticket = &models.Ticket{
			EventID:       req.EventID,
			UserID:        req.UserID,
			TicketCode:    ticketCode,
			PaymentStatus: "paid",
			InvoiceID:     "",
			UMAAddress:    req.UMAAddress,
		}

		if err := h.ticketRepo.Create(ticket); err != nil {
			h.logger.Error("Failed to create free ticket", "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create ticket")
			return
		}
	} else {
		// Paid event - create per-ticket invoice
		if err := h.umaService.ValidateUMAAddress(req.UMAAddress); err != nil {
			middleware.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Invalid UMA address: %v", err))
			return
		}

		// 1. Create ticket with pending payment (no invoice yet)
		ticket = &models.Ticket{
			EventID:       req.EventID,
			UserID:        req.UserID,
			TicketCode:    ticketCode,
			PaymentStatus: "pending",
			UMAAddress:    req.UMAAddress,
		}

		if err := h.ticketRepo.Create(ticket); err != nil {
			h.logger.Error("Failed to create ticket", "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create ticket")
			return
		}

		// 2. Create a new invoice for this ticket (using buyer's UMA address)
		description := fmt.Sprintf("Ticket #%d for %s", ticket.ID, event.Title)

		invoice, err := h.umaService.CreateTicketInvoice(req.UMAAddress, event.PriceSats, description)
		if err != nil {
			h.logger.Error("Failed to create ticket invoice", "ticket_id", ticket.ID, "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create payment invoice")
			return
		}

		// 3. Store the invoice in uma_request_invoices with ticket_id
		ticketInvoice = &models.UMARequestInvoice{
			EventID:     &req.EventID,
			TicketID:    &ticket.ID,
			InvoiceID:   invoice.ID,
			PaymentHash: invoice.PaymentHash,
			Bolt11:      invoice.Bolt11,
			AmountSats:  invoice.AmountSats,
			Status:      invoice.Status,
			UMAAddress:  req.UMAAddress,
			Description: description,
			ExpiresAt:   invoice.ExpiresAt,
		}

		if err := h.umaRepo.Create(ticketInvoice); err != nil {
			h.logger.Error("Failed to save ticket invoice", "ticket_id", ticket.ID, "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to save payment invoice")
			return
		}

		// 4. Create payment record with the new bolt11
		payment := &models.Payment{
			TicketID:  ticket.ID,
			InvoiceID: invoice.Bolt11,
			Amount:    invoice.AmountSats,
			Status:    "pending",
		}

		if err := h.paymentRepo.Create(payment); err != nil {
			h.logger.Error("Failed to create payment record", "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to create payment record")
			return
		}

		// 5. Update ticket's invoice_id
		ticket.InvoiceID = invoice.ID
		if err := h.ticketRepo.Update(ticket); err != nil {
			h.logger.Error("Failed to update ticket invoice_id", "ticket_id", ticket.ID, "error", err)
		}

		h.logger.Info("Ticket created with per-ticket invoice",
			"ticket_id", ticket.ID,
			"invoice_id", invoice.ID,
			"uma_address", req.UMAAddress)

		// Pay the invoice: try NWC first, then SendUMARequest
		go func() {
			// Look up user's NWC connection
			nwcConn, err := h.nwcRepo.GetByUserID(req.UserID)
			if err != nil {
				h.logger.Warn("Failed to look up NWC connection", "user_id", req.UserID, "error", err)
			}

			if nwcConn != nil {
				h.logger.Info("NWC connection found, attempting payment", "ticket_id", ticket.ID, "user_id", req.UserID)
				if err := h.umaService.PayWithNWC(invoice.Bolt11, nwcConn.ConnectionURI); err != nil {
					h.logger.Warn("NWC pay_invoice failed, falling back", "ticket_id", ticket.ID, "error", err)
				} else {
					h.logger.Info("NWC payment initiated successfully", "ticket_id", ticket.ID)
					return
				}
			} else {
				h.logger.Info("No NWC connection found for user, using UMA Request", "user_id", req.UserID)
			}

			// Send UMA Request to buyer's VASP
			callbackURL := fmt.Sprintf("https://api.%s/uma/payreq/%d", h.domain, ticket.ID)
			if err := h.umaService.SendUMARequest(req.UMAAddress, event.PriceSats, callbackURL); err != nil {
				h.logger.Error("SendUMARequest failed, marking ticket as failed",
					"ticket_id", ticket.ID, "error", err)
				// Mark ticket and payment as failed
				_ = h.ticketRepo.UpdatePaymentStatus(ticket.ID, "failed")
				_ = h.paymentRepo.UpdateStatus(payment.ID, "failed")
			}
		}()
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

	// Add per-ticket invoice information for paid events
	if event.PriceSats > 0 && ticketInvoice != nil {
		response["uma_request"] = map[string]interface{}{
			"invoice_id":   ticketInvoice.InvoiceID,
			"bolt11":       ticketInvoice.Bolt11,
			"amount_sats":  ticketInvoice.AmountSats,
			"payment_hash": ticketInvoice.PaymentHash,
			"uma_address":  ticketInvoice.UMAAddress,
			"description":  ticketInvoice.Description,
			"expires_at":   ticketInvoice.ExpiresAt,
			"status":       "pending",
		}
		response["payment_required"] = true
	}

	var message string
	if event.PriceSats == 0 {
		message = "Free ticket created successfully"
	} else {
		message = "Ticket purchase initiated successfully"
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
		"event": map[string]interface{}{
			"title":      event.Title,
			"start_time": event.StartTime,
		},
	}

	// Only fetch payment information for tickets that actually have payments
	if ticket.PaymentStatus == "pending" || ticket.PaymentStatus == "paid" {
		// For free tickets (price 0), we don't need to fetch payment records
		// For paid tickets, fetch the payment record
		if event.PriceSats > 0 {
			payment, err := h.paymentRepo.GetByTicketID(ticketID)
			if err != nil {
				h.logger.Error("Failed to fetch payment", "ticket_id", ticketID, "error", err)
				middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch payment")
				return
			}

			statusResponse["payment"] = map[string]interface{}{
				"status":      payment.Status,
				"amount_sats": payment.Amount,
				"invoice_id":  payment.InvoiceID,
			}
		} else {
			// Free ticket - no payment record needed
			statusResponse["payment"] = map[string]interface{}{
				"status":      "paid",
				"amount_sats": 0,
				"invoice_id":  nil,
			}
		}
	} else {
		// For other statuses (cancelled, etc.)
		statusResponse["payment"] = map[string]interface{}{
			"status":      ticket.PaymentStatus,
			"amount_sats": 0,
			"invoice_id":  nil,
		}
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
				"id":          payment.ID,
				"status":      payment.Status,
				"amount_sats": payment.Amount,
				"invoice_id":  payment.InvoiceID,
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
