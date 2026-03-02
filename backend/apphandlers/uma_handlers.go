package apphandlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/uma-universal-money-address/uma-go-sdk/uma"
	umaprotocol "github.com/uma-universal-money-address/uma-go-sdk/uma/protocol"

	"tickets-by-uma/repositories"
	umaservices "tickets-by-uma/services"
)

// UmaHandlers handles UMA protocol endpoints (payreq callbacks, pubkey, configuration).
type UmaHandlers struct {
	paymentRepo          repositories.PaymentRepository
	umaService           umaservices.UMAService
	logger               *slog.Logger
	domain               string
	umaSigningPrivKeyHex string
}

func NewUmaHandlers(
	paymentRepo repositories.PaymentRepository,
	umaService umaservices.UMAService,
	logger *slog.Logger,
	domain string,
	umaSigningPrivKeyHex string,
) *UmaHandlers {
	return &UmaHandlers{
		paymentRepo:          paymentRepo,
		umaService:           umaService,
		logger:               logger,
		domain:               domain,
		umaSigningPrivKeyHex: umaSigningPrivKeyHex,
	}
}

// HandleUmaPayreq handles UMA pay request callbacks from the buyer's VASP.
// POST /uma/payreq/{ticket_id}
//
// When our app sends a UMA Invoice to test.uma.me and the buyer approves,
// test.uma.me POSTs a pay request here. We create a Lightning invoice and return it.
func (h *UmaHandlers) HandleUmaPayreq(w http.ResponseWriter, r *http.Request) {
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

	// Reuse the bolt11 already created during ticket purchase.
	bolt11 := payment.InvoiceID
	metadata := fmt.Sprintf(`[["text/plain","Ticket purchase at %s"]]`, h.domain)
	invoiceCreator := existingInvoiceCreator{bolt11: bolt11}

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
		response := map[string]interface{}{
			"pr":     bolt11,
			"routes": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	conversionRate := 1000.0 // msats per sat
	decimals := 0
	exchangeFees := int64(0)
	payeeIdentifier := "$tickets@" + h.domain

	// Decode signing key for UMA compliance data
	signingKey, err := hex.DecodeString(h.umaSigningPrivKeyHex)
	if err != nil {
		h.logger.Error("Failed to decode UMA signing key", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	payreqResponse, err := uma.GetPayReqResponse(
		*payreq,
		invoiceCreator,
		metadata,
		payreq.ReceivingCurrencyCode,
		&decimals,
		&conversionRate,
		&exchangeFees,
		nil,              // receiverChannelUtxos
		nil,              // receiverNodePubKey
		nil,              // utxoCallback
		nil,              // payeeData
		&signingKey,      // receivingVaspPrivateKey
		&payeeIdentifier, // payeeIdentifier
		nil,              // disposable
		nil,              // successAction
	)
	if err != nil {
		h.logger.Error("Failed to create payreq response", "error", err)
		response := map[string]interface{}{
			"pr":     bolt11,
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
func (h *UmaHandlers) HandlePubKeyRequest(w http.ResponseWriter, r *http.Request) {
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
// GET/POST /.well-known/uma-configuration
func (h *UmaHandlers) HandleUmaConfiguration(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if h.domain == "localhost" || h.domain == "localhost:8080" {
		scheme = "http"
	}
	response := map[string]interface{}{
		"uma_major_versions":   uma.GetSupportedMajorVersions(),
		"uma_request_endpoint": fmt.Sprintf("%s://%s/uma/payreq/0", scheme, h.domain),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// existingInvoiceCreator returns a pre-existing bolt11 instead of creating a new one.
type existingInvoiceCreator struct {
	bolt11 string
}

func (c existingInvoiceCreator) CreateInvoice(amountMsats int64, metadata string, receiverIdentifier *string) (*string, error) {
	return &c.bolt11, nil
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
