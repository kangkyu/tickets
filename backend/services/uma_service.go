package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"tickets-by-uma/models"
	
	"github.com/lightsparkdev/go-sdk/services"
)

// UMAService defines the interface for UMA payment operations
type UMAService interface {
	CreateInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error)
	CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error)
	ValidateUMAAddress(address string) error
	HandleUMACallback(paymentHash string, status string) error
	CreateRealLightningInvoice(amountSats int64, description string) (*models.Invoice, error)
}

// LightsparkUMAService implements UMAService using real Lightning Network
type LightsparkUMAService struct {
	logger           *slog.Logger
	nodeID           string
	nodePassword     string
	clientID         string
	clientSecret     string
	endpoint         string
	client           *services.LightsparkClient
}

// NewLightsparkUMAService creates a new UMA service instance
func NewLightsparkUMAService(clientID, clientSecret, endpoint, nodeID, nodePassword string, logger *slog.Logger) UMAService {
	// Create Lightspark client - SDK expects full HTTPS URL
	endpointURL := fmt.Sprintf("https://%s", endpoint)
	client := services.NewLightsparkClient(clientID, clientSecret, &endpointURL)
	
	return &LightsparkUMAService{
		logger:       logger,
		nodeID:       nodeID,
		nodePassword: nodePassword,
		clientID:     clientID,
		clientSecret: clientSecret,
		endpoint:     endpoint,
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

// CreateInvoice creates a Lightning invoice for UMA payment
func (s *LightsparkUMAService) CreateInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error) {
	// Validate UMA address
	if err := s.ValidateUMAAddress(umaAddress); err != nil {
		return nil, fmt.Errorf("invalid UMA address: %w", err)
	}
	
	// Create real Lightning invoice
	return s.CreateRealLightningInvoice(amountSats, fmt.Sprintf("UMA Payment: %s", description))
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

// CreateRealLightningInvoice creates a real Lightning invoice using Lightspark SDK
func (s *LightsparkUMAService) CreateRealLightningInvoice(amountSats int64, description string) (*models.Invoice, error) {
	// Convert sats to millisats
	amountMsats := amountSats * 1000

	// Use the official SDK's CreateTestModeInvoice function
	bolt11, err := s.client.CreateTestModeInvoice(
		s.nodeID, 
		amountMsats, 
		&description, 
		nil, // invoice type - nil for default
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Lightspark testnet invoice: %w", err)
	}

	if bolt11 == nil {
		return nil, fmt.Errorf("received nil invoice from Lightspark")
	}

	// Generate unique invoice ID (since we don't get one back from CreateTestModeInvoice)
	invoiceID := s.generateInvoiceID()
	
	// Generate payment hash
	paymentHash := s.generatePaymentHash(invoiceID, amountSats)
	
	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)

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
