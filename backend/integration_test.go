package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"tickets-by-uma/config"
	"tickets-by-uma/models"
	"tickets-by-uma/server"
	"tickets-by-uma/services"
)

// Test configuration
const (
	testDBURL = "postgres://postgres:password@localhost:5432/tickets_uma_test?sslmode=disable"
)

// Test fixture data
var testUsers = []models.CreateUserRequest{
	{Email: "admin@test.com", Name: "Admin User", Password: "password123"},
	{Email: "buyer@test.com", Name: "Test Buyer", Password: "password123"},
	{Email: "freebuyer@test.com", Name: "Free Ticket Buyer", Password: "password123"},
	{Email: "user1@test.com", Name: "Regular User 1", Password: "password123"},
	{Email: "user2@test.com", Name: "Regular User 2", Password: "password123"},
}

// TestServer holds test server and dependencies
type TestServer struct {
	server     *server.Server
	db         *sqlx.DB
	httpServer *httptest.Server
	users      map[string]*models.User // Cache of created fixture users
	tokens     map[string]string       // Cache of JWT tokens
}

// MockUMAService implements services.UMAService for testing
type MockUMAService struct {
	logger *slog.Logger
}

func NewMockUMAService(logger *slog.Logger) services.UMAService {
	return &MockUMAService{logger: logger}
}

func (m *MockUMAService) ValidateUMAAddress(address string) error {
	if address == "" {
		return fmt.Errorf("UMA address cannot be empty")
	}
	return nil
}

func (m *MockUMAService) CreateUMARequest(umaAddress string, amountSats int64, description string, isAdmin bool) (*models.Invoice, error) {
	m.logger.Info("Mock CreateUMARequest called",
		"uma_address", umaAddress,
		"amount_sats", amountSats,
		"description", description,
		"is_admin", isAdmin)
	return &models.Invoice{
		ID:          "test-invoice-123",
		PaymentHash: "test-payment-hash-456",
		Bolt11:      "lntb10000n1p3testmockinvoiceforsimulationpurposes1234567890abcdefghijklmnopqrstuvwxyz",
		AmountSats:  amountSats,
		Status:      "pending",
		ExpiresAt:   timePtr(time.Now().Add(time.Hour)),
	}, nil
}

func (m *MockUMAService) CreateTicketInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error) {
	m.logger.Info("Mock CreateTicketInvoice called",
		"uma_address", umaAddress,
		"amount_sats", amountSats,
		"description", description)
	return m.CreateUMARequest(umaAddress, amountSats, description, false)
}

func (m *MockUMAService) SimulateIncomingPayment(bolt11 string) error {
	m.logger.Info("Mock SimulateIncomingPayment called", "bolt11_prefix", bolt11[:20]+"...")
	return nil
}

func (m *MockUMAService) SendUMARequest(buyerUMA string, amountSats int64, callbackURL string) error {
	m.logger.Info("Mock SendUMARequest called", "buyer_uma", buyerUMA, "amount_sats", amountSats, "callback_url", callbackURL)
	return nil
}

func (m *MockUMAService) GetUMASigningCertChain() string {
	return ""
}

func (m *MockUMAService) GetUMAEncryptionCertChain() string {
	return ""
}

func (m *MockUMAService) SendPaymentToInvoice(bolt11 string) (*models.PaymentResult, error) {
	m.logger.Info("Mock SendPaymentToInvoice called", "bolt11", bolt11[:20]+"...")
	return &models.PaymentResult{
		PaymentID:  "test-outgoing-payment-123",
		Status:     "success", 
		AmountSats: 1000,
		Message:    "Mock payment sent successfully - webhook will be triggered",
	}, nil
}

func (m *MockUMAService) CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error) {
	return &models.PaymentStatus{
		InvoiceID:   invoiceID,
		Status:      "paid",
		AmountSats:  1000,
		PaymentHash: "test-hash",
	}, nil
}

func (m *MockUMAService) GetNodeBalance() (*models.NodeBalance, error) {
	return &models.NodeBalance{
		TotalBalanceSats:     100000,
		AvailableBalanceSats: 90000,
		NodeID:               "test-node-123",
		Status:               "ready",
	}, nil
}

func (m *MockUMAService) PayWithNWC(bolt11 string, nwcConnectionURI string) error {
	m.logger.Info("Mock PayWithNWC called", "bolt11_prefix", bolt11[:20]+"...", "nwc_uri_prefix", nwcConnectionURI[:20]+"...")
	return nil
}

func (m *MockUMAService) HandleUMACallback(paymentHash string, status string) error {
	return nil
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}

// setupTestServer creates a test server with test database
func setupTestServer(t *testing.T) *TestServer {
	// Load test configuration
	testConfig := &config.Config{
		Port:                   "8080",
		DatabaseURL:            testDBURL,
		LightsparkClientID:     "test-client-id",
		LightsparkClientSecret: "test-client-secret",
		LightsparkNodeID:       "test-node-id",
		LightsparkNodePassword: "test-node-password",
		JWTSecret:              "test-jwt-secret",
		AdminEmails:            []string{"admin@test.com"},
		Domain:                 "test.localhost",
	}

	// Connect to test database
	db, err := sqlx.Connect("postgres", testConfig.DatabaseURL)
	if err != nil {
		t.Skip("Test database not available:", err)
	}

	// Clean database before tests
	cleanDatabase(t, db)

	// Create logger with INFO level to see mock service calls
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create mock UMA service
	mockUMAService := NewMockUMAService(logger)

	// Create server with mock service
	srv := server.NewServer(db, logger, testConfig)
	srv.SetUMAService(mockUMAService)

	// Create HTTP test server
	httpServer := httptest.NewServer(srv.Router())

	ts := &TestServer{
		server:     srv,
		db:         db,
		httpServer: httpServer,
		users:      make(map[string]*models.User),
		tokens:     make(map[string]string),
	}

	// Load fixture users
	ts.loadUserFixtures(t)

	return ts
}

// cleanDatabase truncates all tables for clean test state
func cleanDatabase(t *testing.T, db *sqlx.DB) {
	tables := []string{"payments", "tickets", "events", "nwc_connections", "users", "uma_request_invoices"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Could not truncate table %s: %v", table, err)
		}
	}
}

// teardownTestServer cleans up test resources
func (ts *TestServer) teardown() {
	ts.httpServer.Close()
	ts.db.Close()
}

// loadUserFixtures creates fixture users and caches them with tokens
func (ts *TestServer) loadUserFixtures(t *testing.T) {
	for _, userReq := range testUsers {
		// Create user
		userJSON, _ := json.Marshal(userReq)
		resp, err := http.Post(ts.httpServer.URL+"/api/users", "application/json", bytes.NewBuffer(userJSON))
		if err != nil {
			t.Fatalf("Failed to create fixture user %s: %v", userReq.Email, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201 for fixture user %s creation, got %d", userReq.Email, resp.StatusCode)
		}

		// Get user data from response
		var createResp models.SuccessResponse
		json.NewDecoder(resp.Body).Decode(&createResp)
		userData := createResp.Data.(map[string]interface{})

		// Convert to User model
		user := &models.User{
			ID:    int(userData["id"].(float64)),
			Email: userData["email"].(string),
			Name:  userData["name"].(string),
		}

		// Store user in cache
		ts.users[userReq.Email] = user

		// Login user to get token
		loginReq := models.LoginRequest{
			Email:    userReq.Email,
			Password: userReq.Password,
		}

		loginJSON, _ := json.Marshal(loginReq)
		resp, err = http.Post(ts.httpServer.URL+"/api/users/login", "application/json", bytes.NewBuffer(loginJSON))
		if err != nil {
			t.Fatalf("Failed to login fixture user %s: %v", userReq.Email, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200 for fixture user %s login, got %d", userReq.Email, resp.StatusCode)
		}

		var loginResp models.SuccessResponse
		json.NewDecoder(resp.Body).Decode(&loginResp)
		loginData := loginResp.Data.(map[string]interface{})

		token, ok := loginData["token"].(string)
		if !ok {
			t.Fatalf("Expected JWT token for fixture user %s", userReq.Email)
		}

		// Store token in cache
		ts.tokens[userReq.Email] = token
	}
}

// getUser returns a cached fixture user by email
func (ts *TestServer) getUser(email string) *models.User {
	return ts.users[email]
}

// getToken returns a cached JWT token by email
func (ts *TestServer) getToken(email string) string {
	return ts.tokens[email]
}

// getAdminToken returns the admin user's JWT token from fixtures
func (ts *TestServer) getAdminToken() string {
	return ts.getToken("admin@test.com")
}

// Test User Registration and Login
func TestUserRegistrationAndLogin(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	// Test user registration
	user := models.CreateUserRequest{
		Email:    "test@example.com",
		Name:     "Test User",
		Password: "password123",
	}

	userJSON, _ := json.Marshal(user)
	resp, err := http.Post(ts.httpServer.URL+"/api/users", "application/json", bytes.NewBuffer(userJSON))
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var createResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&createResp)

	// Test user login
	loginReq := models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	loginJSON, _ := json.Marshal(loginReq)
	resp, err = http.Post(ts.httpServer.URL+"/api/users/login", "application/json", bytes.NewBuffer(loginJSON))
	if err != nil {
		t.Fatal("Failed to login:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var loginResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&loginResp)
	loginData := loginResp.Data.(map[string]interface{})

	if loginData["token"] == nil {
		t.Error("Expected JWT token in login response")
	}
}

// Test Event Creation and Listing
func TestEventOperations(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	// Get admin token from fixtures
	adminToken := ts.getAdminToken()

	// Create test event with admin authentication
	event := models.CreateEventRequest{
		Title:       "Test Concert",
		Description: "A test concert event",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(26 * time.Hour),
		Capacity:    100,
		PriceSats:   5000,
		StreamURL:   "https://example.com/stream",
	}

	eventJSON, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", ts.httpServer.URL+"/api/admin/events", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Failed to create event:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test event listing
	resp, err = http.Get(ts.httpServer.URL + "/api/events")
	if err != nil {
		t.Fatal("Failed to list events:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var listResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&listResp)
	events := listResp.Data.([]interface{})

	if len(events) == 0 {
		t.Error("Expected at least one event in listing")
	}
}

// Test Ticket Purchase Flow
func TestTicketPurchaseFlow(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	// Get admin token from fixtures
	adminToken := ts.getAdminToken()

	// First create an event with admin authentication
	event := models.CreateEventRequest{
		Title:       "Ticket Test Event",
		Description: "Event for testing ticket purchase",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(26 * time.Hour),
		Capacity:    50,
		PriceSats:   2000,
		StreamURL:   "https://example.com/stream",
	}

	eventJSON, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", ts.httpServer.URL+"/api/admin/events", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Failed to create event:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 for event creation, got %d", resp.StatusCode)
	}

	var eventResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&eventResp)
	eventData := eventResp.Data.(map[string]interface{})
	eventID := eventData["id"].(float64)

	// No need to pre-create UMA invoice — invoices are now created per ticket during purchase

	// Use fixture buyer user
	buyer := ts.getUser("buyer@test.com")
	buyerID := buyer.ID

	// Purchase ticket using the correct request model
	purchaseReq := models.TicketPurchaseRequest{
		EventID:    int(eventID),
		UserID:     buyerID,
		UMAAddress: "$buyer@example.com",
	}

	purchaseJSON, _ := json.Marshal(purchaseReq)
	resp, err = http.Post(ts.httpServer.URL+"/api/tickets/purchase", "application/json", bytes.NewBuffer(purchaseJSON))
	if err != nil {
		t.Fatal("Failed to purchase ticket:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read the error response for debugging
		var errorResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errorResp)
		t.Errorf("Expected status 201, got %d. Error: %s", resp.StatusCode, errorResp.Message)
		return
	}

	var purchaseResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&purchaseResp)

	if purchaseResp.Data == nil {
		t.Error("Response data is nil")
		return
	}

	ticketData := purchaseResp.Data.(map[string]interface{})

	if ticketData["ticket"] == nil {
		t.Error("Expected ticket data in purchase response")
	}

	if ticketData["uma_request"] == nil {
		t.Error("Expected UMA request data in purchase response")
	}
}

// Test Free Ticket Purchase Flow
func TestFreeTicketPurchaseFlow(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	// Get admin token from fixtures
	adminToken := ts.getAdminToken()

	// Create a very low-cost event (since business logic doesn't allow free events)
	event := models.CreateEventRequest{
		Title:       "Low Cost Test Event",
		Description: "Low cost event for testing ticket purchase",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(26 * time.Hour),
		Capacity:    50,
		PriceSats:   1, // Minimal cost event (1 sat)
		StreamURL:   "https://example.com/stream",
	}

	eventJSON, _ := json.Marshal(event)
	req, _ := http.NewRequest("POST", ts.httpServer.URL+"/api/admin/events", bytes.NewBuffer(eventJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Failed to create free event:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read error response for debugging
		var errorResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errorResp)
		t.Fatalf("Expected status 201 for free event creation, got %d. Error: %s", resp.StatusCode, errorResp.Message)
	}

	var eventResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&eventResp)
	eventData := eventResp.Data.(map[string]interface{})
	eventID := eventData["id"].(float64)

	// No need to pre-create UMA invoice — invoices are now created per ticket during purchase

	// Use fixture free buyer user
	freeBuyer := ts.getUser("freebuyer@test.com")
	buyerID := freeBuyer.ID

	// Purchase low cost ticket
	purchaseReq := models.TicketPurchaseRequest{
		EventID:    int(eventID),
		UserID:     buyerID,
		UMAAddress: "$freebuyer@example.com",
	}

	purchaseJSON, _ := json.Marshal(purchaseReq)
	resp, err = http.Post(ts.httpServer.URL+"/api/tickets/purchase", "application/json", bytes.NewBuffer(purchaseJSON))
	if err != nil {
		t.Fatal("Failed to purchase free ticket:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errorResp models.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errorResp)
		t.Errorf("Expected status 201 for free ticket purchase, got %d. Error: %s", resp.StatusCode, errorResp.Message)
		return
	}

	var purchaseResp models.SuccessResponse
	json.NewDecoder(resp.Body).Decode(&purchaseResp)

	if purchaseResp.Data == nil {
		t.Error("Response data is nil")
		return
	}

	ticketData := purchaseResp.Data.(map[string]interface{})

	if ticketData["ticket"] == nil {
		t.Error("Expected ticket data in purchase response")
	}

	// Low cost tickets should have UMA request data
	if ticketData["uma_request"] == nil {
		t.Error("Expected UMA request data in low cost ticket purchase response")
	}
}

//// Test Payment Status Check
//func TestPaymentStatusCheck(t *testing.T) {
//	ts := setupTestServer(t)
//	defer ts.teardown()
//
//	// Test payment status endpoint
//	resp, err := http.Get(ts.httpServer.URL + "/api/payments/test-invoice-123/status")
//	if err != nil {
//		t.Fatal("Failed to check payment status:", err)
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		t.Errorf("Expected status 200, got %d", resp.StatusCode)
//	}
//
//	var statusResp models.SuccessResponse
//	json.NewDecoder(resp.Body).Decode(&statusResp)
//	statusData := statusResp.Data.(map[string]interface{})
//
//	if statusData["status"] != "paid" {
//		t.Error("Expected payment status to be 'paid'")
//	}
//}
//
//// Test Webhook Handling
//func TestPaymentWebhook(t *testing.T) {
//	ts := setupTestServer(t)
//	defer ts.teardown()
//
//	// Test webhook payload - use map since PaymentWebhook doesn't exist
//	webhook := map[string]interface{}{
//		"payment_hash": "test-payment-hash-456",
//		"status":       "paid",
//		"amount_sats":  5000,
//		"invoice_id":   "test-invoice-123",
//	}
//
//	webhookJSON, _ := json.Marshal(webhook)
//	resp, err := http.Post(ts.httpServer.URL+"/api/webhooks/payment", "application/json", bytes.NewBuffer(webhookJSON))
//	if err != nil {
//		t.Fatal("Failed to send webhook:", err)
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		t.Errorf("Expected status 200, got %d", resp.StatusCode)
//	}
//}

//// Test Node Balance Endpoint
//func TestNodeBalance(t *testing.T) {
//	ts := setupTestServer(t)
//	defer ts.teardown()
//
//	resp, err := http.Get(ts.httpServer.URL + "/api/admin/balance")
//	if err != nil {
//		t.Fatal("Failed to get node balance:", err)
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		t.Errorf("Expected status 200, got %d", resp.StatusCode)
//	}
//
//	var balanceResp models.SuccessResponse
//	json.NewDecoder(resp.Body).Decode(&balanceResp)
//	balanceData := balanceResp.Data.(map[string]interface{})
//
//	if balanceData["total_balance_sats"] == nil {
//		t.Error("Expected total_balance_sats in response")
//	}
//}

// Test Error Cases
func TestErrorCases(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.teardown()

	// Test invalid JSON
	resp, err := http.Post(ts.httpServer.URL+"/api/users", "application/json", bytes.NewBuffer([]byte("invalid json")))
	if err != nil {
		t.Fatal("Failed to send invalid JSON:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}

	// Test non-existent event
	resp, err = http.Get(ts.httpServer.URL + "/api/events/99999")
	if err != nil {
		t.Fatal("Failed to get non-existent event:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent event, got %d", resp.StatusCode)
	}

	// Test invalid UMA address
	invalidTicket := models.PurchaseTicketRequest{
		EventID:    1,
		UserEmail:  "test@example.com",
		UserName:   "Test User",
		UMAAddress: "invalid-address",
	}

	ticketJSON, _ := json.Marshal(invalidTicket)
	resp, err = http.Post(ts.httpServer.URL+"/api/tickets/purchase", "application/json", bytes.NewBuffer(ticketJSON))
	if err != nil {
		t.Fatal("Failed to purchase ticket with invalid UMA:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UMA address, got %d", resp.StatusCode)
	}
}

// Benchmark test for concurrent ticket purchases
func BenchmarkConcurrentTicketPurchases(b *testing.B) {
	ts := setupTestServer(&testing.T{})
	defer ts.teardown()

	// Create test event
	event := models.CreateEventRequest{
		Title:       "Benchmark Event",
		Description: "Event for benchmark testing",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(26 * time.Hour),
		Capacity:    1000,
		PriceSats:   1000,
		StreamURL:   "https://example.com/stream",
	}

	eventJSON, _ := json.Marshal(event)
	http.Post(ts.httpServer.URL+"/api/admin/events", "application/json", bytes.NewBuffer(eventJSON))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			purchaseReq := models.PurchaseTicketRequest{
				EventID:    1,
				UserEmail:  fmt.Sprintf("user%d@example.com", i),
				UserName:   fmt.Sprintf("User %d", i),
				UMAAddress: fmt.Sprintf("$user%d@example.com", i),
			}

			purchaseJSON, _ := json.Marshal(purchaseReq)
			resp, _ := http.Post(ts.httpServer.URL+"/api/tickets/purchase", "application/json", bytes.NewBuffer(purchaseJSON))
			if resp != nil {
				resp.Body.Close()
			}
			i++
		}
	})
}
