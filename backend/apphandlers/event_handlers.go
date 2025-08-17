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

type EventHandlers struct {
	eventRepo   repositories.EventRepository
	paymentRepo repositories.PaymentRepository
	umaService  services.UMAService
	logger      *slog.Logger
}

func NewEventHandlers(
	eventRepo repositories.EventRepository,
	paymentRepo repositories.PaymentRepository,
	umaService services.UMAService,
	logger *slog.Logger,
) *EventHandlers {
	return &EventHandlers{
		eventRepo:   eventRepo,
		paymentRepo: paymentRepo,
		umaService:  umaService,
		logger:      logger,
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
	
	event, err := h.eventRepo.GetByID(eventID)
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
	
	// No need to pre-create invoices with UMA Request pattern
	// Invoices will be created on-demand when users purchase tickets
	
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

