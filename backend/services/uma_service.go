package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"tickets-by-uma/models"
)

// UMAService defines the interface for UMA payment operations
type UMAService interface {
	CreateInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error)
	CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error)
	ValidateUMAAddress(address string) error
	HandleUMACallback(paymentHash string, status string) error
}

// LightsparkUMAService implements UMAService using Lightspark and UMA Go SDKs
type LightsparkUMAService struct {
	logger           *slog.Logger
	nodeID           string
	apiToken         string
	endpoint         string
}

// NewLightsparkUMAService creates a new UMA service instance
func NewLightsparkUMAService(apiToken, endpoint, nodeID string, logger *slog.Logger) UMAService {
	return &LightsparkUMAService{
		logger:   logger,
		nodeID:   nodeID,
		apiToken: apiToken,
		endpoint: endpoint,
	}
}

// ValidateUMAAddress validates a UMA address format
func (s *LightsparkUMAService) ValidateUMAAddress(address string) error {
	if address == "" {
		return errors.New("UMA address cannot be empty")
	}

	// Basic UMA address validation
	// UMA addresses typically follow the format: $uma@domain.com
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

	s.logger.Info("Validated UMA address", 
		"identifier", identifier, 
		"domain", domain)
	
	return nil
}

// CreateUMAPaymentRequest creates a UMA payment request
func (s *LightsparkUMAService) CreateUMAPaymentRequest(umaAddress string, amountSats int64, description string) (map[string]interface{}, error) {
	// Validate UMA address
	if err := s.ValidateUMAAddress(umaAddress); err != nil {
		return nil, err
	}

	// Create UMA payment request structure
	payRequest := map[string]interface{}{
		"receiver_address": umaAddress,
		"sending_amount":   amountSats * 1000, // Convert to msats
		"currency":         "SAT",
		"comment":          description,
		"payer_data": map[string]interface{}{
			"identifier": umaAddress[1:], // Remove $ prefix
		},
	}

	return payRequest, nil
}

// CreateInvoice creates a Lightning invoice for UMA payment
func (s *LightsparkUMAService) CreateInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error) {
	s.logger.Info("Creating UMA invoice", 
		"uma_address", umaAddress, 
		"amount_sats", amountSats)

	// Step 1: Validate UMA address
	if err := s.ValidateUMAAddress(umaAddress); err != nil {
		return nil, fmt.Errorf("invalid UMA address: %w", err)
	}
	
	// Step 2: Create UMA payment request
	payRequest, err := s.CreateUMAPaymentRequest(umaAddress, amountSats, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create UMA payment request: %w", err)
	}
	
	// Step 3: Generate metadata and receiver hashes for UMA compliance
	_ = s.generateMetadataHash(description)
	_ = s.generateReceiverHash(umaAddress)
	
	// Step 4: Create invoice (in a real implementation, this would use Lightspark SDK)
	// For now, we'll create a mock invoice structure
	invoiceID := s.generateInvoiceID()
	paymentHash := s.generatePaymentHash(umaAddress, amountSats)
	
	// Mock Bolt11 invoice (in real implementation, this would come from Lightspark)
	bolt11 := fmt.Sprintf("lnbc%d0p1p%s", amountSats, paymentHash[:20])
	
	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)
	
	s.logger.Info("Created UMA invoice", 
		"invoice_id", invoiceID,
		"payment_hash", paymentHash,
		"uma_request", payRequest)

	return &models.Invoice{
		ID:          invoiceID,
		PaymentHash: paymentHash,
		Bolt11:      bolt11,
		AmountSats:  amountSats,
		Status:      "pending",
		ExpiresAt:   &expiresAt,
	}, nil
}

// CheckPaymentStatus checks the status of a payment
func (s *LightsparkUMAService) CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error) {
	s.logger.Info("Checking payment status", "invoice_id", invoiceID)
	
	// In a real implementation, this would query Lightspark or your Lightning node
	// For now, return a mock status
	return &models.PaymentStatus{
		InvoiceID:   invoiceID,
		Status:      "pending",
		AmountSats:  0, // Would be fetched from actual invoice
		PaymentHash: "", // Would be fetched from actual invoice
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
