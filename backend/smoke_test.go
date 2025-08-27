package main

import (
	"testing"

	"tickets-by-uma/models"
	"tickets-by-uma/services"
)

// TestBasicStructures tests that our basic structures and interfaces work
func TestBasicStructures(t *testing.T) {
	// Test that models can be created
	user := &models.User{
		ID:    1,
		Email: "test@example.com",
		Name:  "Test User",
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", user.Email)
	}

	// Test that request structs work
	purchaseReq := &models.PurchaseTicketRequest{
		EventID:    1,
		UserEmail:  "buyer@example.com",
		UserName:   "Test Buyer",
		UMAAddress: "$buyer@example.com",
	}

	if purchaseReq.EventID != 1 {
		t.Errorf("Expected EventID 1, got %d", purchaseReq.EventID)
	}

	// Test UMA service interface exists
	var _ services.UMAService = (*services.LightsparkUMAService)(nil)

	t.Log("All basic structures work correctly")
}

// TestUMAServiceCreation tests that we can create a UMA service
func TestUMAServiceCreation(t *testing.T) {
	// This should not panic
	service := services.NewLightsparkUMAService("", "", "", "", nil)

	if service == nil {
		t.Error("Expected service to be created")
	}

	// Test validation works
	err := service.ValidateUMAAddress("$test@example.com")
	if err != nil {
		t.Errorf("Expected valid UMA address to pass validation, got error: %v", err)
	}

	err = service.ValidateUMAAddress("invalid")
	if err == nil {
		t.Error("Expected invalid UMA address to fail validation")
	}
}
