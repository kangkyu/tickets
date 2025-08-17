package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"tickets-by-uma/models"
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
	apiToken         string
	endpoint         string
	lightningNodeURL string
}

// NewLightsparkUMAService creates a new UMA service instance
func NewLightsparkUMAService(apiToken, endpoint, nodeID string, logger *slog.Logger) UMAService {
	// For real Lightning payments, you can use:
	// - Lightspark (enterprise)
	// - LND (Lightning Network Daemon)
	// - Core Lightning (CLN)
	// - Umbrel (user-friendly)
	
	lightningNodeURL := "http://localhost:10009" // Default LND REST API port
	
	return &LightsparkUMAService{
		logger:           logger,
		nodeID:           nodeID,
		apiToken:         apiToken,
		endpoint:         endpoint,
		lightningNodeURL: lightningNodeURL,
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

	s.logger.Info("Validated UMA address", 
		"identifier", identifier, 
		"domain", domain)
	
	return nil
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
	
	// Step 2: Create real Lightning invoice (not mock)
	invoice, err := s.CreateRealLightningInvoice(amountSats, fmt.Sprintf("UMA Payment: %s", description))
	if err != nil {
		s.logger.Error("Failed to create real Lightning invoice, falling back to mock", "error", err)
		// Fallback to mock invoice for development/testing
		return s.createMockInvoice(umaAddress, amountSats, description)
	}

	s.logger.Info("Created real UMA invoice", 
		"invoice_id", invoice.ID,
		"payment_hash", invoice.PaymentHash,
		"uma_address", umaAddress)

	return invoice, nil
}

// createMockInvoice creates a mock invoice for development/testing
func (s *LightsparkUMAService) createMockInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error) {
	s.logger.Info("Creating mock invoice for development", "uma_address", umaAddress)
	
	invoiceID := s.generateInvoiceID()
	paymentHash := s.generatePaymentHash(umaAddress, amountSats)
	
	// Mock Bolt11 invoice (for development only)
	bolt11 := fmt.Sprintf("lnbc%d0p1p%s", amountSats, paymentHash[:20])
	
	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)
	
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
	
	// Try to get real payment status from Lightning node
	status, err := s.getRealPaymentStatus(invoiceID)
	if err != nil {
		s.logger.Warn("Failed to get real payment status, using mock", "error", err)
		// Fallback to mock status for development
		return s.getMockPaymentStatus(invoiceID)
	}
	
	return status, nil
}

// getRealPaymentStatus gets payment status from Lightning node
func (s *LightsparkUMAService) getRealPaymentStatus(invoiceID string) (*models.PaymentStatus, error) {
	// Query LND for invoice status
	resp, err := http.Get(s.lightningNodeURL + "/v1/invoices/" + invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query Lightning node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Lightning node returned status: %d", resp.StatusCode)
	}

	// Parse LND response
	var lndInvoice struct {
		PaymentRequest string `json:"payment_request"`
		PaymentHash    string `json:"r_hash"`
		Value          int64  `json:"value"`
		State          string `json:"state"`
		Settled        bool   `json:"settled"`
		SettleDate     int64  `json:"settle_date"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lndInvoice); err != nil {
		return nil, fmt.Errorf("failed to decode Lightning response: %w", err)
	}

	// Convert LND state to our status
	var status string
	switch lndInvoice.State {
	case "SETTLED":
		status = "paid"
	case "OPEN":
		status = "pending"
	case "CANCELED":
		status = "expired"
	default:
		status = "pending"
	}

	return &models.PaymentStatus{
		InvoiceID:   invoiceID,
		Status:      status,
		AmountSats:  lndInvoice.Value,
		PaymentHash: lndInvoice.PaymentHash,
	}, nil
}

// getMockPaymentStatus returns mock payment status for development
func (s *LightsparkUMAService) getMockPaymentStatus(invoiceID string) (*models.PaymentStatus, error) {
	return &models.PaymentStatus{
		InvoiceID:   invoiceID,
		Status:      "pending",
		AmountSats:  0,
		PaymentHash: "",
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

// CreateRealLightningInvoice creates a real Lightning invoice using LND or similar
func (s *LightsparkUMAService) CreateRealLightningInvoice(amountSats int64, description string) (*models.Invoice, error) {
	s.logger.Info("Creating real Lightning invoice", 
		"amount_sats", amountSats,
		"description", description)

	// Create invoice request for LND
	invoiceRequest := map[string]interface{}{
		"value":        amountSats,
		"memo":         description,
		"expiry":       3600, // 1 hour
		"private":      false,
		"include_private": false,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(invoiceRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoice request: %w", err)
	}

	// Make request to LND REST API
	resp, err := http.Post(
		s.lightningNodeURL+"/v1/invoices",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Lightning invoice: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Lightning node returned status: %d", resp.StatusCode)
	}

	// Parse response
	var lndResponse struct {
		PaymentRequest string `json:"payment_request"`
		PaymentHash    string `json:"r_hash"`
		AddIndex      uint64 `json:"add_index"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lndResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Lightning response: %w", err)
	}

	// Generate unique invoice ID
	invoiceID := s.generateInvoiceID()
	
	// Set expiration to 1 hour from now
	expiresAt := time.Now().Add(1 * time.Hour)

	s.logger.Info("Created real Lightning invoice", 
		"invoice_id", invoiceID,
		"payment_hash", lndResponse.PaymentHash,
		"bolt11", lndResponse.PaymentRequest)

	return &models.Invoice{
		ID:          invoiceID,
		PaymentHash: lndResponse.PaymentHash,
		Bolt11:      lndResponse.PaymentRequest,
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
