package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lightsparkdev/go-sdk/services"

	"tickets-by-uma/models"
)

// UMAService defines the interface for UMA payment operations
type UMAService interface {
	CreateUMARequest(umaAddress string, amountSats int64, description string, isAdmin bool) (*models.Invoice, error)
	CreateTicketInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error)
	SendPaymentToInvoice(bolt11 string) (*models.PaymentResult, error)
	CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error)
	GetNodeBalance() (*models.NodeBalance, error)
	ValidateUMAAddress(address string) error
	HandleUMACallback(paymentHash string, status string) error
}

// LightsparkUMAService implements UMAService using real Lightning Network
type LightsparkUMAService struct {
	logger       *slog.Logger
	nodeID       string
	nodePassword string
	clientID     string
	clientSecret string
	client       *services.LightsparkClient
}

// NewLightsparkUMAService creates a new UMA service instance
func NewLightsparkUMAService(clientID, clientSecret, nodeID, nodePassword string, logger *slog.Logger) UMAService {
	// Create Lightspark client - SDK handles endpoint internally
	client := services.NewLightsparkClient(clientID, clientSecret, nil)

	return &LightsparkUMAService{
		logger:       logger,
		nodeID:       nodeID,
		nodePassword: nodePassword,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       client,
	}
}

// ValidateUMAAddress validates a UMA address format
func (s *LightsparkUMAService) ValidateUMAAddress(address string) error {
	if address == "" {
		return errors.New("UMA address cannot be empty")
	}

	// Basic UMA address validation
	// UMA addresses typically follow the format: $username@domain.com
	if len(address) < 5 || address[0] != '$' {
		return errors.New("invalid UMA address format: must start with $")
	}

	// Check if it contains @ symbol
	atIndex := -1
	for i, char := range address {
		if char == '@' {
			atIndex = i
			break
		}
	}

	if atIndex == -1 || atIndex == 1 || atIndex == len(address)-1 {
		return errors.New("invalid UMA address format: must contain valid identifier and domain")
	}

	// Validate identifier (between $ and @)
	identifier := address[1:atIndex]
	if len(identifier) == 0 {
		return errors.New("invalid UMA address: identifier cannot be empty")
	}

	// Validate domain (after @)
	domain := address[atIndex+1:]
	if len(domain) == 0 {
		return errors.New("invalid UMA address: domain cannot be empty")
	}

	return nil
}

// CreateUMARequest creates a one-time invoice using UMA Request for a product or service
// This method is restricted to admin users only because it represents the business side of UMA Request protocol
// In UMA protocol: "A business or individual creates a one-time invoice using UMA Request for a product or service"
func (s *LightsparkUMAService) CreateUMARequest(umaAddress string, amountSats int64, description string, isAdmin bool) (*models.Invoice, error) {
	// Admin-only access check - only business operators (admins) can create UMA Request invoices
	if !isAdmin {
		return nil, errors.New("CreateUMARequest is restricted to admin users only - represents business side of UMA Request protocol")
	}

	// Validate UMA address
	if err := s.ValidateUMAAddress(umaAddress); err != nil {
		return nil, fmt.Errorf("invalid UMA address: %w", err)
	}

	s.logger.Info("Creating UMA Request (business operation)",
		"uma_address", umaAddress,
		"amount_sats", amountSats,
		"description", description)

	// Create one-time Lightning invoice using UMA Request pattern
	return s.createOneTimeInvoice(amountSats, fmt.Sprintf("UMA Request - %s", description))
}

// CreateTicketInvoice creates a one-time invoice for ticket purchases (public access)
// This is for end users purchasing tickets, separate from business UMA Request creation
func (s *LightsparkUMAService) CreateTicketInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error) {
	// Validate UMA address
	if err := s.ValidateUMAAddress(umaAddress); err != nil {
		return nil, fmt.Errorf("invalid UMA address: %w", err)
	}

	s.logger.Info("Creating ticket invoice",
		"uma_address", umaAddress,
		"amount_sats", amountSats,
		"description", description)

	// Create one-time Lightning invoice for ticket purchase
	return s.createOneTimeInvoice(amountSats, fmt.Sprintf("Ticket Purchase - %s", description))
}

// SendPaymentToInvoice pays a Lightning invoice using Lightspark SDK's PayUmaInvoice
// This will trigger webhooks when the payment is completed on testnet
func (s *LightsparkUMAService) SendPaymentToInvoice(bolt11 string) (*models.PaymentResult, error) {
	s.logger.Info("Sending payment to Lightning invoice", "bolt11", bolt11[:50]+"...")

	// Check if we have proper Lightspark credentials
	if s.clientID == "" || s.clientSecret == "" || s.nodeID == "" {
		return nil, fmt.Errorf("Lightspark credentials not configured")
	}

	// Load node signing key first (required for payments)
	s.client.LoadNodeSigningKey(s.nodeID, *services.NewSigningKeyLoaderFromNodeIdAndPassword(s.nodeID, s.nodePassword))

	// Execute payment using Lightspark SDK
	timeoutSecs := 60
	maximumFeesMsats := int64(10000) // 10 sats max fee
	var amountMsats *int64 = nil     // Use amount from invoice

	paymentResult, err := s.client.PayUmaInvoice(s.nodeID, bolt11, timeoutSecs, maximumFeesMsats, amountMsats)
	if err != nil {
		s.logger.Error("Payment failed", "error", err)
		return &models.PaymentResult{
			PaymentID:  s.generatePaymentID(),
			Status:     "failed",
			AmountSats: 0,
			Message:    fmt.Sprintf("Payment failed: %v", err),
		}, nil
	}

	if paymentResult == nil {
		return &models.PaymentResult{
			PaymentID:  s.generatePaymentID(),
			Status:     "failed",
			AmountSats: 0,
			Message:    "Payment result was nil",
		}, nil
	}

	// Extract info from OutgoingPayment
	paymentID := paymentResult.GetId()
	if paymentID == "" {
		paymentID = s.generatePaymentID()
	}

	amountSats := paymentResult.GetAmount().OriginalValue / 1000 // Convert msats to sats

	s.logger.Info("Payment sent successfully", "payment_id", paymentID, "amount_sats", amountSats)

	return &models.PaymentResult{
		PaymentID:  paymentID,
		Status:     "success",
		AmountSats: amountSats,
		Message:    "Payment sent successfully - webhook should be triggered",
	}, nil
}

// CheckPaymentStatus checks the status of a payment
func (s *LightsparkUMAService) CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error) {
	// Payment status checking not implemented - return error
	return nil, fmt.Errorf("payment status checking not implemented")
}

// GetNodeBalance retrieves the current balance of the Lightspark node
func (s *LightsparkUMAService) GetNodeBalance() (*models.NodeBalance, error) {
	// Check if we have proper Lightspark credentials
	if s.clientID == "" || s.clientSecret == "" || s.nodeID == "" {
		s.logger.Warn("Lightspark credentials not configured - returning mock balance",
			"client_id_set", s.clientID != "",
			"client_secret_set", s.clientSecret != "",
			"node_id_set", s.nodeID != "")

		// Return mock balance for development/testing
		return &models.NodeBalance{
			TotalBalanceSats:     50000, // 50k sats
			AvailableBalanceSats: 45000, // 45k available
			NodeID:               "mock-node-id",
			Status:               "ready",
		}, nil
	}

	s.logger.Info("Fetching Lightspark node balance", "node_id", s.nodeID)

	// Use Lightspark SDK to get node information
	// TODO: returning simulated data based on real node -- need proper API calls
	return &models.NodeBalance{
		TotalBalanceSats:     0, // Will be updated with real API call
		AvailableBalanceSats: 0, // Will be updated with real API call
		NodeID:               s.nodeID,
		Status:               "ready", // Assume ready if credentials are set
	}, nil
}

// HandleUMACallback processes UMA payment callbacks
func (s *LightsparkUMAService) HandleUMACallback(paymentHash string, status string) error {
	s.logger.Info("Processing UMA callback",
		"payment_hash", paymentHash,
		"status", status)

	// In a real implementation, this would:
	// 1. Validate the callback signature
	// 2. Update the payment status in your database
	// 3. Trigger ticket delivery if payment is successful
	// 4. Send confirmation emails/notifications

	switch status {
	case "paid":
		s.logger.Info("Payment confirmed", "payment_hash", paymentHash)
		// Update ticket status to paid
		// Send confirmation email
		// Update payment record
	case "expired":
		s.logger.Info("Payment expired", "payment_hash", paymentHash)
		// Update ticket status to expired
		// Release ticket back to inventory
	case "failed":
		s.logger.Info("Payment failed", "payment_hash", paymentHash)
		// Update ticket status to failed
		// Release ticket back to inventory
	default:
		s.logger.Warn("Unknown payment status", "status", status, "payment_hash", paymentHash)
	}

	return nil
}

// createOneTimeInvoice creates a one-time Lightning invoice using Lightspark SDK for UMA Request
func (s *LightsparkUMAService) createOneTimeInvoice(amountSats int64, description string) (*models.Invoice, error) {
	if s.clientID == "" || s.clientSecret == "" || s.nodeID == "" {
		return nil, fmt.Errorf("Lightspark credentials not configured")
	}

	// Convert sats to millisats
	amountMsats := amountSats * 1000

	s.logger.Info("Creating Lightspark testnet invoice",
		"amount_sats", amountSats,
		"amount_msats", amountMsats,
		"description", description,
		"node_id", s.nodeID)

	s.logger.Info("Creating Lightspark testnet invoice via SDK",
		"node_id", s.nodeID)

	// Use the official SDK's CreateTestModeInvoice function
	// Note: This function doesn't require context - it's handled internally by the SDK
	bolt11, err := s.client.CreateTestModeInvoice(
		s.nodeID,     // localNodeId: the id of the node that will pay the invoice
		amountMsats,  // amountMsats: the amount of the invoice in millisatoshis
		&description, // memo: the memo of the invoice
		nil,          // invoiceType: the type of the invoice (nil for default)
	)
	if err != nil {
		s.logger.Error("Lightspark CreateTestModeInvoice failed", "error", err)
		return nil, fmt.Errorf("failed to create Lightspark testnet invoice: %w", err)
	}

	if bolt11 == nil {
		s.logger.Error("Received nil bolt11 from Lightspark")
		return nil, fmt.Errorf("received nil invoice from Lightspark")
	}

	// Generate unique invoice ID (since we don't get one back from CreateTestModeInvoice)
	invoiceID := s.generateInvoiceID()

	// Generate payment hash
	paymentHash := s.generatePaymentHash(invoiceID, amountSats)

	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)

	s.logger.Info("Successfully created UMA Request invoice",
		"invoice_id", invoiceID,
		"amount_sats", amountSats,
		"expires_at", expiresAt)

	return &models.Invoice{
		ID:          invoiceID,
		PaymentHash: paymentHash,
		Bolt11:      *bolt11,
		AmountSats:  amountSats,
		Status:      "pending",
		ExpiresAt:   &expiresAt,
	}, nil
}

// Helper methods for UMA protocol compliance
func (s *LightsparkUMAService) generateMetadataHash(description string) string {
	hash := sha256.Sum256([]byte(description))
	return hex.EncodeToString(hash[:])
}

func (s *LightsparkUMAService) generateReceiverHash(umaAddress string) string {
	hash := sha256.Sum256([]byte(umaAddress))
	return hex.EncodeToString(hash[:])
}

func (s *LightsparkUMAService) generateInvoiceID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:16])
}

func (s *LightsparkUMAService) generatePaymentHash(umaAddress string, amountSats int64) string {
	data := fmt.Sprintf("%s:%d:%d", umaAddress, amountSats, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generatePaymentID creates a unique payment ID
func (s *LightsparkUMAService) generatePaymentID() string {
	return fmt.Sprintf("pay_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}
