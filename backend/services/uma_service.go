package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"tickets-by-uma/models"

	"github.com/lightsparkdev/go-sdk/services"
)

// UMAService defines the interface for UMA payment operations
type UMAService interface {
	CreateUMARequest(umaAddress string, amountSats int64, description string) (*models.Invoice, error)
	CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error)
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
func (s *LightsparkUMAService) CreateUMARequest(umaAddress string, amountSats int64, description string) (*models.Invoice, error) {
	// Validate UMA address
	if err := s.ValidateUMAAddress(umaAddress); err != nil {
		return nil, fmt.Errorf("invalid UMA address: %w", err)
	}

	s.logger.Info("Creating UMA Request",
		"uma_address", umaAddress,
		"amount_sats", amountSats,
		"description", description)

	// Create one-time Lightning invoice using UMA Request pattern
	return s.createOneTimeInvoice(amountSats, fmt.Sprintf("UMA Request - %s", description))
}

// CheckPaymentStatus checks the status of a payment
func (s *LightsparkUMAService) CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error) {
	// Payment status checking not implemented - return error
	return nil, fmt.Errorf("payment status checking not implemented")
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

// createHardcodedTestInvoice creates a hardcoded test invoice for development/testing
func (s *LightsparkUMAService) createHardcodedTestInvoice(amountSats int64, description string) (*models.Invoice, error) {
	s.logger.Info("Creating hardcoded test invoice for development",
		"amount_sats", amountSats,
		"description", description)

	// Generate unique invoice ID
	invoiceID := s.generateInvoiceID()

	// Generate payment hash
	paymentHash := s.generatePaymentHash(invoiceID, amountSats)

	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)

	// Create a hardcoded test bolt11 invoice
	// This is a mock invoice for development purposes
	bolt11 := fmt.Sprintf("lntb%d0n1p%spp5%s", 
		amountSats, 
		strings.Repeat("0", 10), 
		strings.Repeat("a", 50))

	s.logger.Info("Created hardcoded test invoice",
		"invoice_id", invoiceID,
		"amount_sats", amountSats,
		"bolt11", bolt11[:50] + "...")

	return &models.Invoice{
		ID:          invoiceID,
		PaymentHash: paymentHash,
		Bolt11:      bolt11,
		AmountSats:  amountSats,
		Status:      "pending",
		ExpiresAt:   &expiresAt,
	}, nil
}

// createOneTimeInvoice creates a one-time Lightning invoice using Lightspark SDK for UMA Request
func (s *LightsparkUMAService) createOneTimeInvoice(amountSats int64, description string) (*models.Invoice, error) {
	// Check if we have proper Lightspark credentials
	if s.clientID == "" || s.clientSecret == "" || s.nodeID == "" {
		s.logger.Warn("Lightspark credentials not configured - using hardcoded test invoice", 
			"client_id_set", s.clientID != "",
			"client_secret_set", s.clientSecret != "",
			"node_id_set", s.nodeID != "")
		
		// Return a hardcoded test invoice for development/testing
		return s.createHardcodedTestInvoice(amountSats, description)
	}

	// Convert sats to millisats
	amountMsats := amountSats * 1000

	s.logger.Info("Creating Lightspark testnet invoice",
		"amount_sats", amountSats,
		"amount_msats", amountMsats,
		"description", description,
		"node_id", s.nodeID)

	// Use the official SDK's CreateTestModeInvoice function
	bolt11, err := s.client.CreateTestModeInvoice(
		s.nodeID,
		amountMsats,
		&description,
		nil, // invoice type - nil for default
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
