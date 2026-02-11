package services

import (
	"log/slog"
	"os"
	"testing"
	"time"
)

// Test UMA Service with mock implementation
func TestUMAServiceValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewLightsparkUMAService("", "", "", "", logger)

	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{
			name:    "valid UMA address",
			address: "$user@example.com",
			wantErr: false,
		},
		{
			name:    "empty address",
			address: "",
			wantErr: true,
		},
		{
			name:    "missing $ prefix",
			address: "user@example.com",
			wantErr: true,
		},
		{
			name:    "missing @ symbol",
			address: "$userexample.com",
			wantErr: true,
		},
		{
			name:    "missing domain",
			address: "$user@",
			wantErr: true,
		},
		{
			name:    "missing identifier",
			address: "$@example.com",
			wantErr: true,
		},
		{
			name:    "minimal valid",
			address: "$u@e.c",
			wantErr: false,
		},
		{
			name:    "only $ symbol",
			address: "$",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateUMAAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUMAAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test CreateUMARequest with admin permissions
func TestCreateUMARequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewLightsparkUMAService("", "", "", "", logger)

	// Test admin-only restriction
	_, err := service.CreateUMARequest("$test@example.com", 1000, "Test invoice", false)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}

	// Test with admin user but no credentials (should fail)
	_, err = service.CreateUMARequest("$admin@example.com", 5000, "Admin invoice", true)
	if err == nil {
		t.Error("Expected error for missing Lightspark credentials")
	}

	// Test invalid UMA address
	_, err = service.CreateUMARequest("invalid-address", 1000, "Test", true)
	if err == nil {
		t.Error("Expected error for invalid UMA address")
	}
}

// Test CreateTicketInvoice (public access)
func TestCreateTicketInvoice(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewLightsparkUMAService("", "", "", "", logger)

	// Test without credentials (should fail)
	_, err := service.CreateTicketInvoice("$user@example.com", 2000, "Concert ticket")
	if err == nil {
		t.Error("Expected error for missing Lightspark credentials")
	}

	// Test invalid UMA address
	_, err = service.CreateTicketInvoice("invalid", 1000, "Test")
	if err == nil {
		t.Error("Expected error for invalid UMA address")
	}
}

// Test GetNodeBalance
func TestGetNodeBalance(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Test without credentials (should return mock balance)
	service := NewLightsparkUMAService("", "", "", "", logger)
	balance, err := service.GetNodeBalance()
	if err != nil {
		t.Fatal("Failed to get node balance:", err)
	}

	if balance.NodeID != "mock-node-id" {
		t.Errorf("Expected mock node ID, got '%s'", balance.NodeID)
	}

	if balance.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", balance.Status)
	}

	// Test with fake credentials (should fail with auth error from Lightspark API)
	serviceWithCreds := NewLightsparkUMAService("test-client", "test-secret", "test-node", "test-password", logger)
	_, err = serviceWithCreds.GetNodeBalance()
	if err == nil {
		t.Error("Expected error when using fake credentials, got nil")
	}
}

// Test CheckPaymentStatus
func TestCheckPaymentStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewLightsparkUMAService("", "", "", "", logger)

	// Test unimplemented payment status check
	_, err := service.CheckPaymentStatus("test-invoice-123")
	if err == nil {
		t.Error("Expected error for unimplemented payment status check")
	}

	expectedErrMsg := "payment status checking not implemented"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

// Test HandleUMACallback
func TestHandleUMACallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewLightsparkUMAService("", "", "", "", logger)

	// Test different callback statuses
	testCases := []string{"paid", "expired", "failed", "unknown"}

	for _, status := range testCases {
		t.Run("status_"+status, func(t *testing.T) {
			err := service.HandleUMACallback("test-payment-hash", status)
			if err != nil {
				t.Errorf("Unexpected error for status %s: %v", status, err)
			}
		})
	}
}

// Test helper methods
func TestHelperMethods(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := &LightsparkUMAService{logger: logger}

	// Test generateInvoiceID
	id1 := service.generateInvoiceID()
	id2 := service.generateInvoiceID()

	if id1 == id2 {
		t.Error("Expected different invoice IDs")
	}

	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected invoice ID length 32, got %d", len(id1))
	}

	// Test generatePaymentHash
	hash1 := service.generatePaymentHash("$user@example.com", 1000)
	hash2 := service.generatePaymentHash("$user@example.com", 2000)

	if hash1 == hash2 {
		t.Error("Expected different payment hashes for different amounts")
	}

	if len(hash1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Expected payment hash length 64, got %d", len(hash1))
	}

	// Test generatePaymentID
	paymentID1 := service.generatePaymentID()
	time.Sleep(time.Millisecond) // Ensure different timestamp
	paymentID2 := service.generatePaymentID()

	if paymentID1 == paymentID2 {
		t.Error("Expected different payment IDs")
	}

	// Test generateMetadataHash
	hash1 = service.generateMetadataHash("description1")
	hash2 = service.generateMetadataHash("description2")

	if hash1 == hash2 {
		t.Error("Expected different metadata hashes for different descriptions")
	}

	// Test generateReceiverHash
	hash1 = service.generateReceiverHash("$user1@example.com")
	hash2 = service.generateReceiverHash("$user2@example.com")

	if hash1 == hash2 {
		t.Error("Expected different receiver hashes for different addresses")
	}
}

// Test edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Without credentials, all invoice creation should fail
	service := NewLightsparkUMAService("", "", "", "", logger)

	_, err := service.CreateTicketInvoice("$test@example.com", 0, "Free ticket")
	if err == nil {
		t.Error("Expected error for missing Lightspark credentials")
	}

	_, err = service.CreateTicketInvoice("$test@example.com", 1000, "Test ticket")
	if err == nil {
		t.Error("Expected error for missing Lightspark credentials")
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// Benchmark tests
func BenchmarkValidateUMAAddress(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := NewLightsparkUMAService("", "", "", "", logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.ValidateUMAAddress("$user@example.com")
	}
}

func BenchmarkGeneratePaymentHash(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	service := &LightsparkUMAService{logger: logger}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.generatePaymentHash("$user@example.com", int64(i))
	}
}
