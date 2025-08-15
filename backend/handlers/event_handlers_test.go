package handlers

import (
	"testing"
	"time"

	"tickets-by-uma/models"
)

func TestValidateCreateEventRequest(t *testing.T) {
	handler := &EventHandlers{}

	tests := []struct {
		name    string
		request models.CreateEventRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: models.CreateEventRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartTime:   time.Now().Add(1 * time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
				Capacity:    100,
				PriceSats:   1000,
				StreamURL:   "https://example.com/stream",
			},
			wantErr: false,
		},
		{
			name: "missing title",
			request: models.CreateEventRequest{
				Description: "Test Description",
				StartTime:   time.Now().Add(1 * time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
				Capacity:    100,
				PriceSats:   1000,
			},
			wantErr: true,
		},
		{
			name: "invalid time range",
			request: models.CreateEventRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartTime:   time.Now().Add(2 * time.Hour),
				EndTime:     time.Now().Add(1 * time.Hour),
				Capacity:    100,
				PriceSats:   1000,
			},
			wantErr: true,
		},
		{
			name: "invalid capacity",
			request: models.CreateEventRequest{
				Title:       "Test Event",
				Description: "Test Description",
				StartTime:   time.Now().Add(1 * time.Hour),
				EndTime:     time.Now().Add(2 * time.Hour),
				Capacity:    0,
				PriceSats:   1000,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.validateCreateEventRequest(&tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCreateEventRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
