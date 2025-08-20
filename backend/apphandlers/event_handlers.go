package apphandlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"tickets-by-uma/config"
	"tickets-by-uma/middleware"
	"tickets-by-uma/models"
	"tickets-by-uma/repositories"
	"tickets-by-uma/services"
)

type EventHandlers struct {
	eventRepo   repositories.EventRepository
	paymentRepo repositories.PaymentRepository
	ticketRepo  repositories.TicketRepository
	umaService  services.UMAService
	umaRepo     repositories.UMARequestInvoiceRepository
	logger      *slog.Logger
	config      *config.Config
}

func NewEventHandlers(
	eventRepo repositories.EventRepository,
	paymentRepo repositories.PaymentRepository,
	ticketRepo repositories.TicketRepository,
	umaService services.UMAService,
	umaRepo repositories.UMARequestInvoiceRepository,
	logger *slog.Logger,
	config *config.Config,
) *EventHandlers {
	return &EventHandlers{
		eventRepo:   eventRepo,
		paymentRepo: paymentRepo,
		ticketRepo:  ticketRepo,
		umaService:  umaService,
		umaRepo:     umaRepo,
		logger:      logger,
		config:      config,
	}
}

// HandleGetEvents lists all active events
func (h *EventHandlers) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Fetching events")

	// Parse query parameters
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	events, err := h.eventRepo.GetActive(limit, offset)
	if err != nil {
		h.logger.Error("Failed to fetch events", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch events")
		return
	}

	// Get current user from context (if authenticated)
	var currentUser *models.User
	if user := middleware.GetUserFromContext(r.Context()); user != nil {
		currentUser = user
	}

	// Enrich events with user ticket status
	enrichedEvents := make([]map[string]interface{}, 0, len(events))
	for _, event := range events {
		enrichedEvent := map[string]interface{}{
			"id":          event.ID,
			"title":       event.Title,
			"description": event.Description,
			"start_time":  event.StartTime,
			"end_time":    event.EndTime,
			"capacity":    event.Capacity,
			"price_sats":  event.PriceSats,
			"stream_url":  event.StreamURL,
			"is_active":   event.IsActive,
			"created_at":  event.CreatedAt,
			"updated_at":  event.UpdatedAt,
		}

		// Add UMA invoice information if available
		if event.UMARequestInvoice != nil {
			enrichedEvent["uma_request_invoice"] = map[string]interface{}{
				"invoice_id":   event.UMARequestInvoice.InvoiceID,
				"bolt11":       event.UMARequestInvoice.Bolt11,
				"amount_sats":  event.UMARequestInvoice.AmountSats,
				"payment_hash": event.UMARequestInvoice.PaymentHash,
				"expires_at":   event.UMARequestInvoice.ExpiresAt,
			}
		}

		// Add user ticket status if user is authenticated
		if currentUser != nil {
			hasTicket, err := h.ticketRepo.HasUserTicketForEvent(currentUser.ID, event.ID)
			if err != nil {
				h.logger.Error("Failed to check user ticket status", "user_id", currentUser.ID, "event_id", event.ID, "error", err)
				// Continue without ticket status if there's an error
				enrichedEvent["user_has_ticket"] = false
			} else {
				enrichedEvent["user_has_ticket"] = hasTicket
			}
		} else {
			enrichedEvent["user_has_ticket"] = false
		}

		enrichedEvents = append(enrichedEvents, enrichedEvent)
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Events retrieved successfully",
		Data:    enrichedEvents,
	})
}

// HandleGetEvent gets a specific event by ID
func (h *EventHandlers) HandleGetEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	h.logger.Info("Fetching event", "event_id", eventID)

	// Use GetByIDWithUMAInvoice to include UMA invoice data that the frontend expects
	event, err := h.eventRepo.GetByIDWithUMAInvoice(eventID)
	if err != nil {
		h.logger.Error("Failed to fetch event", "event_id", eventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}

	if event == nil {
		middleware.WriteError(w, http.StatusNotFound, "Event not found")
		return
	}

	// Get current user from context (if authenticated)
	var currentUser *models.User
	if user := middleware.GetUserFromContext(r.Context()); user != nil {
		currentUser = user
	}

	// Enrich event with user ticket status
	enrichedEvent := map[string]interface{}{
		"id":          event.ID,
		"title":       event.Title,
		"description": event.Description,
		"start_time":  event.StartTime,
		"end_time":    event.EndTime,
		"capacity":    event.Capacity,
		"price_sats":  event.PriceSats,
		"stream_url":  event.StreamURL,
		"is_active":   event.IsActive,
		"created_at":  event.CreatedAt,
		"updated_at":  event.UpdatedAt,
	}

	// Add UMA invoice information if available
	if event.UMARequestInvoice != nil {
		enrichedEvent["uma_request_invoice"] = map[string]interface{}{
			"invoice_id":   event.UMARequestInvoice.InvoiceID,
			"bolt11":       event.UMARequestInvoice.Bolt11,
			"amount_sats":  event.UMARequestInvoice.AmountSats,
			"payment_hash": event.UMARequestInvoice.PaymentHash,
			"expires_at":   event.UMARequestInvoice.ExpiresAt,
		}
	}

	// Add user ticket status if user is authenticated
	if currentUser != nil {
		hasTicket, err := h.ticketRepo.HasUserTicketForEvent(currentUser.ID, event.ID)
		if err != nil {
			h.logger.Error("Failed to check user ticket status", "user_id", currentUser.ID, "event_id", event.ID, "error", err)
			// Continue without ticket status if there's an error
			enrichedEvent["user_has_ticket"] = false
		} else {
			enrichedEvent["user_has_ticket"] = hasTicket
		}
	} else {
		enrichedEvent["user_has_ticket"] = false
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Event retrieved successfully",
		Data:    enrichedEvent,
	})
}

// HandleCreateEvent creates a new event (admin only)
func (h *EventHandlers) HandleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := h.validateCreateEventRequest(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Creating new event", "title", req.Title)

	event := &models.Event{
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Capacity:    req.Capacity,
		PriceSats:   req.PriceSats,
		StreamURL:   req.StreamURL,
		IsActive:    true,
	}

	if err := h.eventRepo.Create(event); err != nil {
		h.logger.Error("Failed to create event", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create event")
		return
	}


	h.logger.Info("Event created successfully", "event_id", event.ID)

	middleware.WriteJSON(w, http.StatusCreated, models.SuccessResponse{
		Message: "Event created successfully",
		Data:    event,
	})
}

// HandleUpdateEvent updates an existing event (admin only)
func (h *EventHandlers) HandleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	var req models.UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	h.logger.Info("Updating event", "event_id", eventID)

	// Get existing event
	event, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.logger.Error("Failed to fetch event for update", "event_id", eventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}

	if event == nil {
		middleware.WriteError(w, http.StatusNotFound, "Event not found")
		return
	}

	// Update fields if provided
	if req.Title != nil {
		event.Title = *req.Title
	}
	if req.Description != nil {
		event.Description = *req.Description
	}
	if req.StartTime != nil {
		event.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		event.EndTime = *req.EndTime
	}
	if req.Capacity != nil {
		event.Capacity = *req.Capacity
	}
	if req.PriceSats != nil {
		event.PriceSats = *req.PriceSats
	}
	if req.StreamURL != nil {
		event.StreamURL = *req.StreamURL
	}
	if req.IsActive != nil {
		event.IsActive = *req.IsActive
	}


	// Note: UMA Request invoices are only needed for paid events (price > 0)
	// Free events (price = 0) don't need UMA invoices since tickets are free
	// We don't automatically create invoices for events that don't need them

	// Now save all changes (including UMA invoice information) in a single update
	if err := h.eventRepo.Update(event); err != nil {
		h.logger.Error("Failed to update event", "event_id", eventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to update event")
		return
	}

	h.logger.Info("Event updated successfully", "event_id", eventID)

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Event updated successfully",
		Data:    event,
	})
}

// HandleDeleteEvent deletes an event (admin only)
func (h *EventHandlers) HandleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	h.logger.Info("Deleting event", "event_id", eventID)

	// Check if event exists
	event, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.logger.Error("Failed to fetch event for deletion", "event_id", eventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}

	if event == nil {
		middleware.WriteError(w, http.StatusNotFound, "Event not found")
		return
	}

	if err := h.eventRepo.Delete(eventID); err != nil {
		h.logger.Error("Failed to delete event", "event_id", eventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to delete event")
		return
	}

	h.logger.Info("Event deleted successfully", "event_id", eventID)

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Event deleted successfully",
	})
}


// HandleCreateEventUMAInvoice creates a UMA Request invoice for a specific event (admin only)
func (h *EventHandlers) HandleCreateEventUMAInvoice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid event ID")
		return
	}

	h.logger.Info("Creating UMA Request invoice for event", "event_id", eventID)

	// Get the event
	event, err := h.eventRepo.GetByID(eventID)
	if err != nil {
		h.logger.Error("Failed to fetch event for UMA invoice creation", "event_id", eventID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch event")
		return
	}

	if event == nil {
		middleware.WriteError(w, http.StatusNotFound, "Event not found")
		return
	}

	// Generate UMA address for the event
	umaAddress := "$event@" + h.getDomainFromConfig()
	description := fmt.Sprintf("Event Ticket: %s", event.Title)

	// Create UMA Request invoice for the event
	umaInvoice, err := h.umaService.CreateUMARequest(
		umaAddress,
		event.PriceSats,
		description,
		true, // isAdmin = true for admin endpoints
	)
	if err != nil {
		h.logger.Error("Failed to create UMA Request invoice for event",
			"event_id", eventID,
			"error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create UMA Request invoice")
		return
	}

	// Store the UMA Request invoice in the separate table
	umaInvoiceRecord := &models.UMARequestInvoice{
		EventID:     eventID,
		InvoiceID:   umaInvoice.ID,
		PaymentHash: umaInvoice.PaymentHash,
		Bolt11:      umaInvoice.Bolt11,
		AmountSats:  umaInvoice.AmountSats,
		Status:      umaInvoice.Status,
		UMAAddress:  umaAddress,
		Description: description,
		ExpiresAt:   umaInvoice.ExpiresAt,
	}

	if err := h.umaRepo.Create(umaInvoiceRecord); err != nil {
		h.logger.Error("Failed to save UMA Request invoice to database",
			"event_id", eventID,
			"error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to save UMA Request invoice")
		return
	}

	h.logger.Info("UMA Request invoice created for event",
		"event_id", eventID,
		"invoice_id", umaInvoice.ID,
		"uma_address", umaAddress)

	middleware.WriteJSON(w, http.StatusCreated, models.SuccessResponse{
		Message: "UMA Request invoice created successfully for event",
		Data: map[string]interface{}{
			"event": map[string]interface{}{
				"id":    event.ID,
				"title": event.Title,
			},
			"invoice": map[string]interface{}{
				"id":           umaInvoice.ID,
				"payment_hash": umaInvoice.PaymentHash,
				"bolt11":       umaInvoice.Bolt11,
				"amount_sats":  umaInvoice.AmountSats,
				"status":       umaInvoice.Status,
				"expires_at":   umaInvoice.ExpiresAt,
			},
			"uma_address": umaAddress,
		},
	})
}

// validateCreateEventRequest validates the create event request
func (h *EventHandlers) validateCreateEventRequest(req *models.CreateEventRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if req.StartTime.IsZero() {
		return fmt.Errorf("start time is required")
	}

	if req.EndTime.IsZero() {
		return fmt.Errorf("end time is required")
	}

	if req.StartTime.After(req.EndTime) {
		return fmt.Errorf("start time must be before end time")
	}

	if req.StartTime.Before(time.Now()) {
		return fmt.Errorf("start time cannot be in the past")
	}

	if req.Capacity <= 0 {
		return fmt.Errorf("capacity must be greater than 0")
	}

	if req.PriceSats <= 0 {
		return fmt.Errorf("price must be greater than 0")
	}

	return nil
}

func (h *EventHandlers) getDomainFromConfig() string {
	if h.config == nil {
		return "localhost" // Fallback if config is not available
	}
	return h.config.Domain
}
