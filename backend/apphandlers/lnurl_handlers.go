package apphandlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/lightsparkdev/go-sdk/services"
	"github.com/uma-universal-money-address/uma-go-sdk/uma"
	umaprotocol "github.com/uma-universal-money-address/uma-go-sdk/uma/protocol"

	"tickets-by-uma/repositories"
	umaservices "tickets-by-uma/services"
)

// LnurlHandlers handles LNURL-pay protocol endpoints and UMA payreq callbacks.
type LnurlHandlers struct {
	paymentRepo repositories.PaymentRepository
	umaService  umaservices.UMAService
	lsClient    *services.LightsparkClient
	logger      *slog.Logger
	domain      string
	nodeID      string
}

func NewLnurlHandlers(
	paymentRepo repositories.PaymentRepository,
	umaService umaservices.UMAService,
	lsClient *services.LightsparkClient,
	logger *slog.Logger,
	domain string,
	nodeID string,
) *LnurlHandlers {
	return &LnurlHandlers{
		paymentRepo: paymentRepo,
		umaService:  umaService,
		lsClient:    lsClient,
		logger:      logger,
		domain:      domain,
		nodeID:      nodeID,
	}
}

// HandleLnurlPay handles LNURL-pay initial resolution.
// GET /.well-known/lnurlp/tickets
//
// When a user types $tickets@fanmeeting.org in test.uma.me,
// test.uma.me calls this endpoint to discover payment parameters.
func (h *LnurlHandlers) HandleLnurlPay(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("LNURL-pay resolution request")

	metadata := `[["text/plain","Ticket purchase at fanmeeting.org"]]`
	callbackURL := fmt.Sprintf("https://%s/api/lnurl/callback", h.domain)

	response := map[string]interface{}{
		"callback":    callbackURL,
		"maxSendable": 10000000000, // 10M sats in msats
		"minSendable": 1000,        // 1 sat in msats
		"metadata":    metadata,
		"tag":         "payRequest",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleLnurlCallback handles the LNURL-pay callback.
// GET /api/lnurl/callback?amount={msats}
//
// Finds the oldest pending payment matching the requested amount
// and returns its bolt11 invoice.
func (h *LnurlHandlers) HandleLnurlCallback(w http.ResponseWriter, r *http.Request) {
	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		writeLnurlError(w, "amount parameter is required")
		return
	}

	amountMsats, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amountMsats <= 0 {
		writeLnurlError(w, "invalid amount")
		return
	}

	amountSats := amountMsats / 1000

	h.logger.Info("LNURL-pay callback", "amount_sats", amountSats)

	// Find the oldest pending payment with this amount
	payment, err := h.paymentRepo.GetOldestPendingByAmount(amountSats)
	if err != nil {
		h.logger.Error("Failed to find pending payment", "amount_sats", amountSats, "error", err)
		writeLnurlError(w, "No pending payment found")
		return
	}

	if payment == nil {
		h.logger.Warn("No pending payment for amount", "amount_sats", amountSats)
		writeLnurlError(w, "No pending ticket for this amount")
		return
	}

	h.logger.Info("LNURL-pay callback returning bolt11",
		"payment_id", payment.ID,
		"ticket_id", payment.TicketID,
		"bolt11_prefix", payment.InvoiceID[:min(len(payment.InvoiceID), 30)]+"...")

	response := map[string]interface{}{
		"pr":     payment.InvoiceID,
		"routes": []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUmaPayreq handles UMA pay request callbacks from the buyer's VASP.
// POST /uma/payreq/{ticket_id}
//
// When our app sends a UMA Invoice to test.uma.me and the buyer approves,
// test.uma.me POSTs a pay request here. We create a Lightning invoice and return it.
func (h *LnurlHandlers) HandleUmaPayreq(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticketIDStr := vars["ticket_id"]
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "invalid ticket_id", http.StatusBadRequest)
		return
	}

	h.logger.Info("UMA payreq callback received", "ticket_id", ticketID)

	// Find the pending payment for this ticket
	payment, err := h.paymentRepo.GetByTicketID(ticketID)
	if err != nil || payment == nil {
		h.logger.Error("No payment found for ticket", "ticket_id", ticketID, "error", err)
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	if payment.Status != "pending" {
		h.logger.Warn("Payment not pending", "ticket_id", ticketID, "status", payment.Status)
		http.Error(w, "payment not pending", http.StatusBadRequest)
		return
	}

	// The payment.InvoiceID is the bolt11 — return it to the sender's VASP
	metadata := fmt.Sprintf(`[["text/plain","Ticket purchase at %s"]]`, h.domain)

	expirySecs := int32(600)
	invoiceCreator := LightsparkLnurlInvoiceCreator{
		Client:     h.lsClient,
		NodeId:     h.nodeID,
		ExpirySecs: &expirySecs,
	}

	// Parse the incoming pay request
	requestBody, err := readBody(r)
	if err != nil {
		h.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	payreq, err := uma.ParsePayRequest(requestBody)
	if err != nil {
		h.logger.Error("Failed to parse pay request", "error", err)
		// Fall back: return the existing bolt11 directly
		response := map[string]interface{}{
			"pr":     payment.InvoiceID,
			"routes": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	conversionRate := 1000.0 // msats per sat
	decimals := 0
	exchangeFees := int64(0)

	payreqResponse, err := uma.GetPayReqResponse(
		*payreq,
		invoiceCreator,
		metadata,
		payreq.ReceivingCurrencyCode,
		&decimals,
		&conversionRate,
		&exchangeFees,
		nil, // receiverUtxos
		nil, // receiverNodePubKey
		nil, // utxoCallback
		nil, // payeeData
		nil, // signingPrivateKey
		nil, // receiverIdentifier
		nil, // receiverChannelUtxos
		nil, // disposable
	)
	if err != nil {
		h.logger.Error("Failed to create payreq response", "error", err)
		// Fall back: return the existing bolt11
		response := map[string]interface{}{
			"pr":     payment.InvoiceID,
			"routes": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	h.logger.Info("UMA payreq response sent", "ticket_id", ticketID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payreqResponse)
}

// HandlePubKeyRequest serves UMA public keys for signature verification.
// GET /.well-known/lnurlpubkey
func (h *LnurlHandlers) HandlePubKeyRequest(w http.ResponseWriter, r *http.Request) {
	signingCertChain := h.umaService.GetUMASigningCertChain()
	encryptionCertChain := h.umaService.GetUMAEncryptionCertChain()

	if signingCertChain == "" || encryptionCertChain == "" {
		h.logger.Warn("UMA keys not configured")
		http.Error(w, "UMA keys not configured", http.StatusServiceUnavailable)
		return
	}

	twoWeeksFromNow := time.Now().AddDate(0, 0, 14)
	twoWeeksFromNowSec := twoWeeksFromNow.Unix()
	response, err := uma.GetPubKeyResponse(signingCertChain, encryptionCertChain, &twoWeeksFromNowSec)
	if err != nil {
		h.logger.Error("Failed to get pubkey response", "error", err)
		http.Error(w, "failed to generate pubkey response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUmaConfiguration serves the UMA configuration for this domain.
// POST /.well-known/uma-configuration
func (h *LnurlHandlers) HandleUmaConfiguration(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if h.domain == "localhost" || h.domain == "localhost:8080" {
		scheme = "http"
	}
	response := map[string]interface{}{
		"uma_major_versions":   uma.GetSupportedMajorVersions(),
		"uma_request_endpoint": fmt.Sprintf("%s://%s/uma/request_invoice_payment", scheme, h.domain),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LightsparkLnurlInvoiceCreator implements the InvoiceCreator interface for LNURL invoices
type LightsparkLnurlInvoiceCreator struct {
	Client     *services.LightsparkClient
	NodeId     string
	ExpirySecs *int32
}

func (l LightsparkLnurlInvoiceCreator) CreateInvoice(amountMsats int64, metadata string, receiverIdentifier *string) (*string, error) {
	invoice, err := l.Client.CreateLnurlInvoice(l.NodeId, amountMsats, metadata, l.ExpirySecs)
	if err != nil {
		return nil, err
	}
	return &invoice.Data.EncodedPaymentRequest, nil
}

// SatsCurrency for UMA payreq responses
var SatsCurrency = umaprotocol.Currency{
	Code:                "SAT",
	Name:                "Satoshis",
	Symbol:              "SAT",
	MillisatoshiPerUnit: 1000,
	Convertible: umaprotocol.ConvertibleCurrency{
		MinSendable: 1,
		MaxSendable: 100_000_000,
	},
	Decimals:        0,
	UmaMajorVersion: 1,
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

func writeLnurlError(w http.ResponseWriter, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // LNURL spec: errors still return 200 with status=ERROR
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ERROR",
		"reason": reason,
	})
}
