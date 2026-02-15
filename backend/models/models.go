package models

import (
	"time"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Name         string    `json:"name" db:"name"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Event represents a virtual event
type Event struct {
	ID          int       `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	StartTime   time.Time `json:"start_time" db:"start_time"`
	EndTime     time.Time `json:"end_time" db:"end_time"`
	Capacity    int       `json:"capacity" db:"capacity"`
	PriceSats   int64     `json:"price_sats" db:"price_sats"`
	StreamURL   string    `json:"stream_url" db:"stream_url"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Relationship to UMA Request invoice (can be nil if no invoice exists)
	UMARequestInvoice *UMARequestInvoice `json:"uma_request_invoice,omitempty" db:"-"`
}

// UMARequestInvoice represents a UMA Request invoice for an event
type UMARequestInvoice struct {
	ID          int        `json:"id" db:"id"`
	EventID     *int       `json:"event_id" db:"event_id"`
	TicketID    *int       `json:"ticket_id,omitempty" db:"ticket_id"`
	InvoiceID   string     `json:"invoice_id" db:"invoice_id"`
	PaymentHash string     `json:"payment_hash" db:"payment_hash"`
	Bolt11      string     `json:"bolt11" db:"bolt11"`
	AmountSats  int64      `json:"amount_sats" db:"amount_sats"`
	Status      string     `json:"status" db:"status"`
	UMAAddress  string     `json:"uma_address" db:"uma_address"`
	Description string     `json:"description" db:"description"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// Ticket represents a ticket for an event
type Ticket struct {
	ID            int        `json:"id" db:"id"`
	EventID       int        `json:"event_id" db:"event_id"`
	UserID        int        `json:"user_id" db:"user_id"`
	TicketCode    string     `json:"ticket_code" db:"ticket_code"`
	PaymentStatus string     `json:"payment_status" db:"payment_status"`
	InvoiceID     string     `json:"invoice_id" db:"invoice_id"`
	UMAAddress    string     `json:"uma_address" db:"uma_address"`
	PaidAt        *time.Time `json:"paid_at" db:"paid_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// Payment represents a payment record
type Payment struct {
	ID        int        `json:"id" db:"id"`
	TicketID  int        `json:"ticket_id" db:"ticket_id"`
	InvoiceID string     `json:"invoice_id" db:"invoice_id"`
	Amount    int64      `json:"amount_sats" db:"amount_sats"`
	Status    string     `json:"status" db:"status"`
	Preimage  *string    `json:"preimage,omitempty" db:"preimage"`
	PaidAt    *time.Time `json:"paid_at" db:"paid_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// Invoice represents a Lightning invoice
type Invoice struct {
	ID          string     `json:"id"`
	PaymentHash string     `json:"payment_hash"`
	Bolt11      string     `json:"bolt11"`
	AmountSats  int64      `json:"amount_sats"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// PaymentStatus represents the status of a payment
type PaymentStatus struct {
	InvoiceID   string `json:"invoice_id"`
	Status      string `json:"status"`
	AmountSats  int64  `json:"amount_sats"`
	PaymentHash string `json:"payment_hash,omitempty"`
}

// PaymentResult represents the result of a payment attempt
type PaymentResult struct {
	PaymentID       string  `json:"payment_id"`
	Status          string  `json:"status"` // "success", "failed", "pending"
	AmountSats      int64   `json:"amount_sats"`
	Message         string  `json:"message"`
	TransactionHash *string `json:"transaction_hash,omitempty"`
}

// NodeBalance represents the Lightning node balance information
type NodeBalance struct {
	TotalBalanceSats     int64  `json:"total_balance_sats"`
	AvailableBalanceSats int64  `json:"available_balance_sats"`
	NodeID               string `json:"node_id"`
	Status               string `json:"status"`
}

// NWCConnection represents a stored NWC connection for a user
type NWCConnection struct {
	ID            int        `json:"id" db:"id"`
	UserID        int        `json:"user_id" db:"user_id"`
	ConnectionURI string     `json:"connection_uri" db:"connection_uri"`
	ExpiresAt     *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// StoreNWCConnectionRequest represents a request to store an NWC connection
type StoreNWCConnectionRequest struct {
	NWCConnectionURI string     `json:"nwc_connection_uri"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// TicketPurchaseRequest represents a ticket purchase request
type TicketPurchaseRequest struct {
	EventID    int    `json:"event_id"`
	UserID     int    `json:"user_id"`
	UMAAddress string `json:"uma_address"`
}

// TicketValidationRequest represents a ticket validation request
type TicketValidationRequest struct {
	TicketCode string `json:"ticket_code"`
	EventID    int    `json:"event_id"`
}

// CreateEventRequest represents a request to create an event
type CreateEventRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Capacity    int       `json:"capacity"`
	PriceSats   int64     `json:"price_sats"`
	StreamURL   string    `json:"stream_url"`
}

// UpdateEventRequest represents a request to update an event
type UpdateEventRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Capacity    *int       `json:"capacity,omitempty"`
	PriceSats   *int64     `json:"price_sats,omitempty"`
	StreamURL   *string    `json:"stream_url,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// PurchaseTicketRequest represents a request to purchase a ticket
type PurchaseTicketRequest struct {
	EventID    int    `json:"event_id"`
	UserEmail  string `json:"user_email"`
	UserName   string `json:"user_name"`
	UMAAddress string `json:"uma_address"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// UMACallbackRequest represents a UMA payment callback
type UMACallbackRequest struct {
	PaymentHash string `json:"payment_hash"`
	Status      string `json:"status"`
	InvoiceID   string `json:"invoice_id"`
	AmountSats  int64  `json:"amount_sats,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
	Signature   string `json:"signature,omitempty"`
}
