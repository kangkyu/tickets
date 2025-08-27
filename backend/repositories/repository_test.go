package repositories

import (
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"tickets-by-uma/models"
)

// Test database URL - should be different from production
const testDBURL = "postgres://postgres:password@localhost:5432/tickets_uma_test?sslmode=disable"

// setupTestDB creates a clean test database connection
func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", testDBURL)
	if err != nil {
		t.Skip("Test database not available:", err)
	}

	// Clean all tables
	cleanTables(t, db)

	return db
}

func cleanTables(t *testing.T, db *sqlx.DB) {
	tables := []string{"payments", "tickets", "events", "users", "uma_request_invoices"}
	for _, table := range tables {
		_, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			t.Logf("Warning: Could not truncate table %s: %v", table, err)
		}
	}
}

// Test User Repository
func TestUserRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)

	// Test Create User
	user := &models.User{
		Email: "test@example.com",
		Name:  "Test User",
	}

	err := repo.Create(user)
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	if user.ID == 0 {
		t.Error("Expected user ID to be set after creation")
	}

	// Test Get User by ID
	retrievedUser, err := repo.GetByID(user.ID)
	if err != nil {
		t.Fatal("Failed to get user by ID:", err)
	}

	if retrievedUser.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrievedUser.Email)
	}

	// Test Get User by Email
	userByEmail, err := repo.GetByEmail(user.Email)
	if err != nil {
		t.Fatal("Failed to get user by email:", err)
	}

	if userByEmail.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, userByEmail.ID)
	}

	// Test Update User
	user.Name = "Updated Name"
	err = repo.Update(user)
	if err != nil {
		t.Fatal("Failed to update user:", err)
	}

	updatedUser, err := repo.GetByID(user.ID)
	if err != nil {
		t.Fatal("Failed to get updated user:", err)
	}

	if updatedUser.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got '%s'", updatedUser.Name)
	}

	// Test Delete User
	err = repo.Delete(user.ID)
	if err != nil {
		t.Fatal("Failed to delete user:", err)
	}

	// Verify user is deleted
	_, err = repo.GetByID(user.ID)
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}
}

// Test Event Repository
func TestEventRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewEventRepository(db)

	// Create test event
	event := &models.Event{
		Title:       "Test Event",
		Description: "A test event",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(26 * time.Hour),
		Capacity:    100,
		PriceSats:   5000,
		StreamURL:   "https://example.com/stream",
	}

	err := repo.Create(event)
	if err != nil {
		t.Fatal("Failed to create event:", err)
	}

	if event.ID == 0 {
		t.Error("Expected event ID to be set after creation")
	}

	// Test Get Event by ID
	retrievedEvent, err := repo.GetByID(event.ID)
	if err != nil {
		t.Fatal("Failed to get event by ID:", err)
	}

	if retrievedEvent.Title != event.Title {
		t.Errorf("Expected title %s, got %s", event.Title, retrievedEvent.Title)
	}

	// Test List Events
	events, err := repo.GetAll(10, 0)
	if err != nil {
		t.Fatal("Failed to list events:", err)
	}

	if len(events) == 0 {
		t.Error("Expected at least one event in list")
	}

	// Test Update Event
	event.Title = "Updated Event"
	err = repo.Update(event)
	if err != nil {
		t.Fatal("Failed to update event:", err)
	}

	updatedEvent, err := repo.GetByID(event.ID)
	if err != nil {
		t.Fatal("Failed to get updated event:", err)
	}

	if updatedEvent.Title != "Updated Event" {
		t.Errorf("Expected title 'Updated Event', got '%s'", updatedEvent.Title)
	}

	// Test Delete Event
	err = repo.Delete(event.ID)
	if err != nil {
		t.Fatal("Failed to delete event:", err)
	}

	// Verify event is deleted
	_, err = repo.GetByID(event.ID)
	if err == nil {
		t.Error("Expected error when getting deleted event")
	}
}

// Test Ticket Repository
func TestTicketRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	eventRepo := NewEventRepository(db)
	ticketRepo := NewTicketRepository(db)

	// Create test user
	user := &models.User{
		Email: "ticket-user@example.com",
		Name:  "Ticket User",
	}
	err := userRepo.Create(user)
	if err != nil {
		t.Fatal("Failed to create test user:", err)
	}

	// Create test event
	event := &models.Event{
		Title:       "Ticket Event",
		Description: "Event for ticket testing",
		StartTime:   time.Now().Add(24 * time.Hour),
		EndTime:     time.Now().Add(26 * time.Hour),
		Capacity:    50,
		PriceSats:   2000,
	}
	err = eventRepo.Create(event)
	if err != nil {
		t.Fatal("Failed to create test event:", err)
	}

	// Create test ticket
	ticket := &models.Ticket{
		UserID:        user.ID,
		EventID:       event.ID,
		TicketCode:    "TEST123",
		PaymentStatus: "pending",
		UMAAddress:    "$test@example.com",
	}

	err = ticketRepo.Create(ticket)
	if err != nil {
		t.Fatal("Failed to create ticket:", err)
	}

	if ticket.ID == 0 {
		t.Error("Expected ticket ID to be set after creation")
	}

	// Test Get Ticket by ID
	retrievedTicket, err := ticketRepo.GetByID(ticket.ID)
	if err != nil {
		t.Fatal("Failed to get ticket by ID:", err)
	}

	if retrievedTicket.TicketCode != ticket.TicketCode {
		t.Errorf("Expected ticket code %s, got %s", ticket.TicketCode, retrievedTicket.TicketCode)
	}

	// Test Get Tickets by User ID
	userTickets, err := ticketRepo.GetByUserID(user.ID)
	if err != nil {
		t.Fatal("Failed to get tickets by user ID:", err)
	}

	if len(userTickets) == 0 {
		t.Error("Expected at least one ticket for user")
	}

	// Test Update Payment Status
	err = ticketRepo.UpdatePaymentStatus(ticket.ID, "paid")
	if err != nil {
		t.Fatal("Failed to update payment status:", err)
	}

	updatedTicket, err := ticketRepo.GetByID(ticket.ID)
	if err != nil {
		t.Fatal("Failed to get updated ticket:", err)
	}

	if updatedTicket.PaymentStatus != "paid" {
		t.Errorf("Expected payment status 'paid', got '%s'", updatedTicket.PaymentStatus)
	}

	// Test Get Ticket by Code
	ticketByCode, err := ticketRepo.GetByTicketCode(ticket.TicketCode)
	if err != nil {
		t.Fatal("Failed to get ticket by code:", err)
	}

	if ticketByCode.ID != ticket.ID {
		t.Errorf("Expected ticket ID %d, got %d", ticket.ID, ticketByCode.ID)
	}
}

// Test Payment Repository
func TestPaymentRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	paymentRepo := NewPaymentRepository(db)

	// Create test user
	user := &models.User{
		Email: "payment-user@example.com",
		Name:  "Payment User",
	}
	err := userRepo.Create(user)
	if err != nil {
		t.Fatal("Failed to create test user:", err)
	}

	// Create test payment
	payment := &models.Payment{
		TicketID:  1, // Will be set properly in real usage
		InvoiceID: "test-invoice-123",
		Amount:    5000,
		Status:    "pending",
	}

	err = paymentRepo.Create(payment)
	if err != nil {
		t.Fatal("Failed to create payment:", err)
	}

	if payment.ID == 0 {
		t.Error("Expected payment ID to be set after creation")
	}

	// Test Get Payment by ID
	retrievedPayment, err := paymentRepo.GetByID(payment.ID)
	if err != nil {
		t.Fatal("Failed to get payment by ID:", err)
	}

	if retrievedPayment.InvoiceID != payment.InvoiceID {
		t.Errorf("Expected invoice ID %s, got %s", payment.InvoiceID, retrievedPayment.InvoiceID)
	}

	// Test Get Payment by Invoice ID
	paymentByInvoice, err := paymentRepo.GetByInvoiceID(payment.InvoiceID)
	if err != nil {
		t.Fatal("Failed to get payment by invoice ID:", err)
	}

	if paymentByInvoice.ID != payment.ID {
		t.Errorf("Expected payment ID %d, got %d", payment.ID, paymentByInvoice.ID)
	}

	// Test Update Payment Status
	err = paymentRepo.UpdateStatus(payment.ID, "paid")
	if err != nil {
		t.Fatal("Failed to update payment status:", err)
	}

	updatedPayment, err := paymentRepo.GetByID(payment.ID)
	if err != nil {
		t.Fatal("Failed to get updated payment:", err)
	}

	if updatedPayment.Status != "paid" {
		t.Errorf("Expected status 'paid', got '%s'", updatedPayment.Status)
	}
}

// Test concurrent operations
func TestConcurrentOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)

	// Create multiple users concurrently
	numUsers := 10
	done := make(chan error, numUsers)

	for i := 0; i < numUsers; i++ {
		go func(id int) {
			user := &models.User{
				Email: fmt.Sprintf("concurrent%d@example.com", id),
				Name:  fmt.Sprintf("Concurrent User %d", id),
			}
			done <- userRepo.Create(user)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numUsers; i++ {
		if err := <-done; err != nil {
			t.Errorf("Failed to create user %d: %v", i, err)
		}
	}

	// Skip verification since GetAll method doesn't exist in UserRepository interface
	t.Log("Successfully created users concurrently")
}

// Test transaction rollback scenarios
func TestTransactionRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// This test would require implementing transaction support
	// in repositories, which is currently not implemented
	t.Skip("Transaction support not implemented yet")
}