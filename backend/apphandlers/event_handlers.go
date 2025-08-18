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
	"tickets-by-uma/config"
)

type EventHandlers struct {
	eventRepo   repositories.EventRepository
	paymentRepo repositories.PaymentRepository
	umaService  services.UMAService
	umaRepo     repositories.UMARequestInvoiceRepository
	logger      *slog.Logger
	config      *config.Config
}

func NewEventHandlers(
	eventRepo repositories.EventRepository,
	paymentRepo repositories.PaymentRepository,
	umaService services.UMAService,
	umaRepo repositories.UMARequestInvoiceRepository,
	logger *slog.Logger,
	config *config.Config,
) *EventHandlers {
	return &EventHandlers{
		eventRepo:   eventRepo,
		paymentRepo: paymentRepo,
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
	
	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Events retrieved successfully",
		Data:    events,
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
	
	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Event retrieved successfully",
		Data:    event,
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
	
	// Create UMA Request invoice for this event's tickets (treating tickets as products)
	// This follows UMA protocol: "A business creates a one-time invoice using UMA Request for a product or service"
	// Note: UMA Request invoices are only needed for paid events (price > 0)
	// Free events (price = 0) don't need UMA invoices since tickets are free
	if event.PriceSats > 0 {
		umaAddress := "$event@" + h.getDomainFromConfig() // Generate UMA address for the event
		description := fmt.Sprintf("Event Ticket: %s", event.Title)
		
		umaInvoice, err := h.umaService.CreateUMARequest(
			umaAddress,
			event.PriceSats,
			description,
			true, // isAdmin = true for admin endpoints
		)
		if err != nil {
			h.logger.Error("Failed to create UMA Request invoice for paid event", 
				"event_id", event.ID, 
				"price_sats", event.PriceSats,
				"error", err)
			// Don't fail the event creation, just log the error
			// The event can still be created without the UMA invoice
		} else {
			// Store the UMA Request invoice in the separate table
			umaInvoiceRecord := &models.UMARequestInvoice{
				EventID:     event.ID,
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
					"event_id", event.ID, 
					"error", err)
			}
			
			h.logger.Info("UMA Request invoice created for paid event tickets",
				"event_id", event.ID,
				"invoice_id", umaInvoice.ID,
				"price_sats", event.PriceSats,
				"uma_address", umaAddress)
		}
	} else {
		h.logger.Info("Event created as free event - no UMA Request invoice needed",
			"event_id", event.ID,
			"price_sats", event.PriceSats)
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

	// Handle UMA Request invoice creation/updates before saving
	// Only paid events (price > 0) need UMA Request invoices
	// Free events (price = 0) don't need UMA invoices since tickets are free
	if req.PriceSats != nil && *req.PriceSats != event.PriceSats {
		if *req.PriceSats > 0 {
			// Event is now paid - create UMA Request invoice
			umaAddress := "$event@" + h.getDomainFromConfig() // Generate UMA address for the event
			description := fmt.Sprintf("Event Ticket: %s (Updated)", event.Title)
			
			umaInvoice, err := h.umaService.CreateUMARequest(
				umaAddress,
				*req.PriceSats, // Use the new price
				description,
				true, // isAdmin = true for admin endpoints
			)
			if err != nil {
				h.logger.Error("Failed to create UMA Request invoice for newly paid event", 
					"event_id", eventID, 
					"new_price", *req.PriceSats,
					"error", err)
				// Don't fail the event update, just log the error
			} else {
				// Check if UMA invoice already exists for this event
				existingInvoice, err := h.umaRepo.GetByEventID(eventID)
				if err != nil {
					h.logger.Error("Failed to check existing UMA invoice", 
						"event_id", eventID, 
						"error", err)
				} else if existingInvoice != nil {
					// Update existing invoice
					existingInvoice.InvoiceID = umaInvoice.ID
					existingInvoice.PaymentHash = umaInvoice.PaymentHash
					existingInvoice.Bolt11 = umaInvoice.Bolt11
					existingInvoice.AmountSats = umaInvoice.AmountSats
					existingInvoice.Status = umaInvoice.Status
					existingInvoice.Description = description
					existingInvoice.ExpiresAt = umaInvoice.ExpiresAt
					
					if err := h.umaRepo.Update(existingInvoice); err != nil {
						h.logger.Error("Failed to update existing UMA invoice", 
							"event_id", eventID, 
							"error", err)
					}
				} else {
					// Create new invoice
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
						h.logger.Error("Failed to create new UMA invoice", 
							"event_id", eventID, 
							"error", err)
					}
				}
				
				h.logger.Info("UMA Request invoice created for newly paid event",
					"event_id", eventID,
					"invoice_id", umaInvoice.ID,
					"new_price", *req.PriceSats,
					"uma_address", umaAddress)
			}
		} else if *req.PriceSats == 0 {
			// Event is now free - remove UMA Request invoice if it exists
			existingInvoice, err := h.umaRepo.GetByEventID(eventID)
			if err != nil {
				h.logger.Error("Failed to check existing UMA invoice for removal", 
					"event_id", eventID, 
					"error", err)
			} else if existingInvoice != nil {
				if err := h.umaRepo.Delete(existingInvoice.ID); err != nil {
					h.logger.Error("Failed to delete UMA invoice for now-free event", 
						"event_id", eventID, 
						"error", err)
				} else {
					h.logger.Info("UMA Request invoice removed - event is now free",
						"event_id", eventID,
						"old_price", event.PriceSats,
						"new_price", *req.PriceSats)
				}
			}
		}
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

// HandleCreateUMARequest creates a UMA request for multi-use invoices (admin only)
func (h *EventHandlers) HandleCreateUMARequest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UMAAddress string `json:"uma_address"`
		AmountSats int64  `json:"amount_sats"`
		Description string `json:"description"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Validate request
	if req.UMAAddress == "" {
		middleware.WriteError(w, http.StatusBadRequest, "UMA address is required")
		return
	}
	
	if req.AmountSats <= 0 {
		middleware.WriteError(w, http.StatusBadRequest, "Valid amount in satoshis is required")
		return
	}
	
	if req.Description == "" {
		middleware.WriteError(w, http.StatusBadRequest, "Description is required")
		return
	}
	
	h.logger.Info("Creating UMA Request (admin operation)",
		"uma_address", req.UMAAddress,
		"amount_sats", req.AmountSats,
		"description", req.Description)
	
	// Create UMA Request - admin-only operation
	invoice, err := h.umaService.CreateUMARequest(
		req.UMAAddress,
		req.AmountSats,
		req.Description,
		true, // isAdmin = true for admin endpoints
	)
	if err != nil {
		h.logger.Error("Failed to create UMA request", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	h.logger.Info("UMA Request created successfully",
		"invoice_id", invoice.ID,
		"uma_address", req.UMAAddress)
	
	middleware.WriteJSON(w, http.StatusCreated, models.SuccessResponse{
		Message: "UMA Request created successfully",
		Data: map[string]interface{}{
			"invoice": map[string]interface{}{
				"id":          invoice.ID,
				"payment_hash": invoice.PaymentHash,
				"bolt11":      invoice.Bolt11,
				"amount_sats": invoice.AmountSats,
				"status":      invoice.Status,
				"expires_at":  invoice.ExpiresAt,
			},
			"uma_address": req.UMAAddress,
			"description": req.Description,
		},
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
				"id": event.ID,
				"title": event.Title,
			},
			"invoice": map[string]interface{}{
				"id":          umaInvoice.ID,
				"payment_hash": umaInvoice.PaymentHash,
				"bolt11":      umaInvoice.Bolt11,
				"amount_sats": umaInvoice.AmountSats,
				"status":      umaInvoice.Status,
				"expires_at":  umaInvoice.ExpiresAt,
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

