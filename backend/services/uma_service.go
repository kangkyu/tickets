package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lightsparkdev/go-sdk/objects"
	"github.com/lightsparkdev/go-sdk/services"
	"github.com/uma-universal-money-address/uma-go-sdk/uma"
	umaprotocol "github.com/uma-universal-money-address/uma-go-sdk/uma/protocol"

	"tickets-by-uma/models"
)

// UMAService defines the interface for UMA payment operations
type UMAService interface {
	CreateUMARequest(umaAddress string, amountSats int64, description string, isAdmin bool) (*models.Invoice, error)
	CreateTicketInvoice(umaAddress string, amountSats int64, description string) (*models.Invoice, error)
	SimulateIncomingPayment(bolt11 string) error
	SendUMARequest(buyerUMA string, amountSats int64, callbackURL string) error
	SendPaymentToInvoice(bolt11 string) (*models.PaymentResult, error)
	CheckPaymentStatus(invoiceID string) (*models.PaymentStatus, error)
	GetNodeBalance() (*models.NodeBalance, error)
	ValidateUMAAddress(address string) error
	HandleUMACallback(paymentHash string, status string) error
	GetUMASigningCertChain() string
	GetUMAEncryptionCertChain() string
}

// LightsparkUMAService implements UMAService using real Lightning Network
type LightsparkUMAService struct {
	logger                 *slog.Logger
	nodeID                 string
	nodePassword           string
	clientID               string
	clientSecret           string
	client                 *services.LightsparkClient
	domain                 string
	umaSigningPrivKeyHex   string
	umaSigningCertChain    string
	umaEncryptionPrivKeyHex string
	umaEncryptionCertChain  string
}

// NewLightsparkUMAService creates a new UMA service instance
func NewLightsparkUMAService(clientID, clientSecret, nodeID, nodePassword, domain, umaSigningPrivKeyHex, umaSigningCertChain, umaEncryptionPrivKeyHex, umaEncryptionCertChain string, logger *slog.Logger) UMAService {
	// Create Lightspark client - SDK handles endpoint internally
	client := services.NewLightsparkClient(clientID, clientSecret, nil)

	return &LightsparkUMAService{
		logger:                  logger,
		nodeID:                  nodeID,
		nodePassword:            nodePassword,
		clientID:                clientID,
		clientSecret:            clientSecret,
		client:                  client,
		domain:                  domain,
		umaSigningPrivKeyHex:    umaSigningPrivKeyHex,
		umaSigningCertChain:     umaSigningCertChain,
		umaEncryptionPrivKeyHex: umaEncryptionPrivKeyHex,
		umaEncryptionCertChain:  umaEncryptionCertChain,
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

// SimulateIncomingPayment uses CreateTestModePayment to simulate an external node
// paying our invoice. This triggers the webhook with an IncomingPayment event,
// just like a real buyer paying from their wallet would.
func (s *LightsparkUMAService) SimulateIncomingPayment(bolt11 string) error {
	s.logger.Info("Simulating incoming payment (test mode)", "bolt11_prefix", bolt11[:min(len(bolt11), 50)]+"...")

	if s.nodeID == "" {
		return fmt.Errorf("node ID not configured")
	}

	incomingPayment, err := s.client.CreateTestModePayment(s.nodeID, bolt11, nil)
	if err != nil {
		s.logger.Error("CreateTestModePayment failed", "error", err)
		return fmt.Errorf("failed to simulate payment: %w", err)
	}

	s.logger.Info("Test mode payment simulated successfully",
		"incoming_payment_id", incomingPayment.Id,
		"amount", incomingPayment.Amount,
		"is_uma", incomingPayment.IsUma)

	return nil
}

// GetUMASigningCertChain returns the UMA signing certificate chain
func (s *LightsparkUMAService) GetUMASigningCertChain() string {
	return s.umaSigningCertChain
}

// GetUMAEncryptionCertChain returns the UMA encryption certificate chain
func (s *LightsparkUMAService) GetUMAEncryptionCertChain() string {
	return s.umaEncryptionCertChain
}

// SendUMARequest creates a UMA Invoice and sends it to the buyer's VASP via UMA Request protocol.
// This pushes a payment request to the buyer's wallet (e.g. test.uma.me).
func (s *LightsparkUMAService) SendUMARequest(buyerUMA string, amountSats int64, callbackURL string) error {
	if s.umaSigningPrivKeyHex == "" {
		return fmt.Errorf("UMA signing key not configured")
	}

	signingKey, err := hex.DecodeString(s.umaSigningPrivKeyHex)
	if err != nil {
		return fmt.Errorf("invalid UMA signing key: %w", err)
	}

	s.logger.Info("Sending UMA Request to buyer's VASP",
		"buyer_uma", buyerUMA,
		"amount_sats", amountSats,
		"callback_url", callbackURL)

	// Create UMA Invoice
	twoDaysFromNow := time.Now().Add(48 * time.Hour)
	receiverUMA := "$tickets@" + s.domain

	invoice, err := uma.CreateUmaInvoice(
		receiverUMA,
		uint64(amountSats),
		umaprotocol.InvoiceCurrency{
			Code:     "SAT",
			Decimals: 0,
			Symbol:   "SAT",
			Name:     "Satoshis",
		},
		uint64(twoDaysFromNow.Unix()),
		callbackURL,
		true, // isSubjectToTravelRule
		nil,  // requiredPayerData
		nil,  // commentLength
		nil,  // senderUma
		nil,  // invoiceLimit
		&buyerUMA,
		signingKey,
	)
	if err != nil {
		return fmt.Errorf("failed to create UMA invoice: %w", err)
	}

	invoiceString, err := invoice.ToBech32String()
	if err != nil {
		return fmt.Errorf("failed to encode UMA invoice: %w", err)
	}

	// Discover buyer's VASP domain
	buyerVASPDomain, err := uma.GetVaspDomainFromUmaAddress(buyerUMA)
	if err != nil {
		return fmt.Errorf("failed to get VASP domain from %s: %w", buyerUMA, err)
	}

	// Determine scheme
	scheme := "https://"
	if strings.Contains(buyerVASPDomain, "localhost") {
		scheme = "http://"
	}

	// Fetch VASP's UMA configuration to get uma_request_endpoint
	configURL := scheme + buyerVASPDomain + "/.well-known/uma-configuration"
	s.logger.Info("Fetching VASP configuration", "url", configURL)

	resp, err := http.Get(configURL)
	if err != nil {
		return fmt.Errorf("failed to fetch VASP configuration from %s: %w", configURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read VASP configuration response: %w", err)
	}

	var vaspConfig struct {
		UMARequestEndpoint string `json:"uma_request_endpoint"`
	}
	if err := json.Unmarshal(body, &vaspConfig); err != nil {
		return fmt.Errorf("failed to parse VASP configuration: %w", err)
	}

	if vaspConfig.UMARequestEndpoint == "" {
		return fmt.Errorf("VASP at %s does not have a uma_request_endpoint", buyerVASPDomain)
	}

	s.logger.Info("Sending UMA invoice to VASP",
		"endpoint", vaspConfig.UMARequestEndpoint,
		"invoice_length", len(invoiceString))

	// Send the invoice to the buyer's VASP
	requestBody, err := json.Marshal(map[string]string{
		"invoice": invoiceString,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal invoice request: %w", err)
	}

	resp2, err := http.Post(vaspConfig.UMARequestEndpoint, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to send invoice to VASP: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("VASP rejected invoice (status %d): %s", resp2.StatusCode, string(respBody))
	}

	s.logger.Info("UMA Request sent successfully",
		"buyer_uma", buyerUMA,
		"vasp_domain", buyerVASPDomain)

	return nil
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

	entity, err := s.client.GetEntity(s.nodeID)
	if err != nil {
		s.logger.Error("Failed to get node entity", "error", err)
		return nil, fmt.Errorf("failed to get node entity: %w", err)
	}
	if entity == nil {
		return nil, fmt.Errorf("node entity not found")
	}

	lightsparkNode, ok := (*entity).(objects.LightsparkNode)
	if !ok {
		return nil, fmt.Errorf("entity is not a LightsparkNode")
	}

	status := "unknown"
	if nodeStatus := lightsparkNode.GetStatus(); nodeStatus != nil {
		status = nodeStatus.StringValue()
	}

	var totalSats, availableSats int64
	if balances := lightsparkNode.GetBalances(); balances != nil {
		totalSats = balances.OwnedBalance.OriginalValue / 1000         // msats to sats
		availableSats = balances.AvailableToSendBalance.OriginalValue / 1000
	}

	return &models.NodeBalance{
		TotalBalanceSats:     totalSats,
		AvailableBalanceSats: availableSats,
		NodeID:               s.nodeID,
		Status:               status,
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

// createOneTimeInvoice creates a one-time LNURL Lightning invoice using Lightspark SDK.
// Uses CreateLnurlInvoice so the bolt11 contains a description_hash that matches
// the LNURL metadata, enabling payments via UMA/LNURL-pay resolution.
func (s *LightsparkUMAService) createOneTimeInvoice(amountSats int64, description string) (*models.Invoice, error) {
	if s.clientID == "" || s.clientSecret == "" || s.nodeID == "" {
		return nil, fmt.Errorf("Lightspark credentials not configured")
	}

	amountMsats := amountSats * 1000

	// Format as LNURL metadata so the description_hash in the bolt11
	// matches what LNURL-pay endpoints serve to paying wallets.
	metadata := fmt.Sprintf(`[["text/plain","%s"]]`, description)

	s.logger.Info("Creating LNURL Lightning invoice",
		"amount_sats", amountSats,
		"description", description,
		"node_id", s.nodeID)

	invoice, err := s.client.CreateLnurlInvoice(
		s.nodeID,
		amountMsats,
		metadata,
		nil, // expirySecs (default 1 day)
	)
	if err != nil {
		s.logger.Error("Lightspark CreateLnurlInvoice failed", "error", err)
		return nil, fmt.Errorf("failed to create Lightning invoice: %w", err)
	}

	if invoice == nil {
		return nil, fmt.Errorf("received nil invoice from Lightspark")
	}

	expiresAt := invoice.Data.ExpiresAt

	s.logger.Info("Successfully created LNURL Lightning invoice",
		"invoice_id", invoice.Id,
		"payment_hash", invoice.Data.PaymentHash,
		"amount_sats", amountSats,
		"expires_at", expiresAt)

	return &models.Invoice{
		ID:          invoice.Id,
		PaymentHash: invoice.Data.PaymentHash,
		Bolt11:      invoice.Data.EncodedPaymentRequest,
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
