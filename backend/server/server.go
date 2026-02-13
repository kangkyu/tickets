package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/lightsparkdev/go-sdk/services"

	"tickets-by-uma/apphandlers"
	"tickets-by-uma/config"
	"tickets-by-uma/middleware"
	"tickets-by-uma/repositories"
	uma_services "tickets-by-uma/services"
)

type Server struct {
	db              *sqlx.DB
	logger          *slog.Logger
	config          *config.Config
	userRepo        repositories.UserRepository
	eventRepo       repositories.EventRepository
	ticketRepo      repositories.TicketRepository
	paymentRepo     repositories.PaymentRepository
	umaRepo         repositories.UMARequestInvoiceRepository
	nwcRepo         repositories.NWCConnectionRepository
	umaService      uma_services.UMAService
	lightsparkClient *services.LightsparkClient
	router          *mux.Router
	userHandlers    *apphandlers.UserHandlers
	eventHandlers   *apphandlers.EventHandlers
	ticketHandlers  *apphandlers.TicketHandlers
	paymentHandlers *apphandlers.PaymentHandlers
	lnurlHandlers   *apphandlers.LnurlHandlers
}

func NewServer(db *sqlx.DB, logger *slog.Logger, config *config.Config) *Server {
	s := &Server{
		db:     db,
		logger: logger,
		config: config,
		router: mux.NewRouter(),
	}

	// Initialize repositories
	s.userRepo = repositories.NewUserRepository(db)
	s.eventRepo = repositories.NewEventRepository(db)
	s.ticketRepo = repositories.NewTicketRepository(db)
	s.paymentRepo = repositories.NewPaymentRepository(db)
	s.umaRepo = repositories.NewUMARequestInvoiceRepository(db)
	s.nwcRepo = repositories.NewNWCConnectionRepository(db)

	// Initialize Lightspark client
	s.lightsparkClient = services.NewLightsparkClient(config.LightsparkClientID, config.LightsparkClientSecret, nil)

	// Initialize UMA service
	s.umaService = uma_services.NewLightsparkUMAService(
		config.LightsparkClientID,
		config.LightsparkClientSecret,
		config.LightsparkNodeID,
		config.LightsparkNodePassword,
		config.Domain,
		config.UMASigningPrivKeyHex,
		config.UMASigningCertChain,
		config.UMAEncryptionPrivKeyHex,
		config.UMAEncryptionCertChain,
		logger,
	)

	// Initialize handlers
	s.initializeHandlers()

	s.setupRoutes()
	return s
}

// SetUMAService allows setting a custom UMA service (useful for testing)
func (s *Server) SetUMAService(umaService uma_services.UMAService) {
	s.umaService = umaService
	// Re-initialize handlers with new service
	s.initializeHandlers()
	// Re-setup routes since handlers changed
	s.router = mux.NewRouter()
	s.setupRoutes()
}

// Router returns the router (useful for testing)
func (s *Server) Router() *mux.Router {
	return s.router
}

func (s *Server) setupRoutes() {
	// Add CORS middleware to main router (covers all endpoints)
	s.router.Use(s.corsMiddleware)

	// Add logging middleware
	s.router.Use(s.loggingMiddleware)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// LNURL-pay resolution endpoint for $tickets@fanmeeting.org
	s.router.HandleFunc("/.well-known/lnurlp/tickets", s.lnurlHandlers.HandleLnurlPay).Methods("GET", "OPTIONS")

	// UMA protocol endpoints
	s.router.HandleFunc("/.well-known/lnurlpubkey", s.lnurlHandlers.HandlePubKeyRequest).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/.well-known/uma-configuration", s.lnurlHandlers.HandleUmaConfiguration).Methods("POST", "GET", "OPTIONS")
	s.router.HandleFunc("/uma/payreq/{ticket_id:[0-9]+}", s.lnurlHandlers.HandleUmaPayreq).Methods("POST", "GET", "OPTIONS")

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()

	// CORS middleware is already applied to main router, no need to apply again
	// api.Use(s.corsMiddleware)

	// User routes (no auth required)
	api.HandleFunc("/users", s.userHandlers.HandleCreateUser).Methods("POST", "OPTIONS")
	api.HandleFunc("/users/login", s.userHandlers.HandleLogin).Methods("POST", "OPTIONS")
	api.HandleFunc("/users/{id:[0-9]+}", s.userHandlers.HandleGetUser).Methods("GET", "OPTIONS")

	// Event routes (public)
	api.HandleFunc("/events", s.eventHandlers.HandleGetEvents).Methods("GET", "OPTIONS")
	api.HandleFunc("/events/{id:[0-9]+}", s.eventHandlers.HandleGetEvent).Methods("GET", "OPTIONS")

	// Ticket routes (public for purchase, auth for others)
	api.HandleFunc("/tickets/purchase", s.ticketHandlers.HandlePurchaseTicket).Methods("POST", "OPTIONS")
	api.HandleFunc("/tickets/{id:[0-9]+}/status", s.ticketHandlers.HandleTicketStatus).Methods("GET", "OPTIONS")
	api.HandleFunc("/tickets/validate", s.ticketHandlers.HandleValidateTicket).Methods("POST", "OPTIONS")
	api.HandleFunc("/tickets/uma-callback", s.ticketHandlers.HandleUMAPaymentCallback).Methods("POST", "OPTIONS")

	// LNURL-pay callback (no auth required - called by paying wallets)
	api.HandleFunc("/lnurl/callback", s.lnurlHandlers.HandleLnurlCallback).Methods("GET", "OPTIONS")

	// Payment webhook (no auth required)
	api.HandleFunc("/webhooks/payment", s.paymentHandlers.HandlePaymentWebhook).Methods("POST", "OPTIONS")

	// Protected routes (require authentication)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware(s.config.JWTSecret))

	// Protected user routes
	protected.HandleFunc("/users/me", s.userHandlers.HandleGetCurrentUser).Methods("GET", "OPTIONS")
	protected.HandleFunc("/users/me/nwc-connection", s.userHandlers.HandleStoreNWCConnection).Methods("POST", "OPTIONS")
	protected.HandleFunc("/users/{id:[0-9]+}", s.userHandlers.HandleUpdateUser).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/users/{id:[0-9]+}", s.userHandlers.HandleDeleteUser).Methods("DELETE", "OPTIONS")

	// Protected ticket routes
	protected.HandleFunc("/users/{user_id:[0-9]+}/tickets", s.ticketHandlers.HandleGetUserTickets).Methods("GET", "OPTIONS")

	// Protected payment routes
	protected.HandleFunc("/payments/{invoice_id}/status", s.paymentHandlers.HandlePaymentStatus).Methods("GET", "OPTIONS")

	// Admin routes (require authentication and admin privileges)
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.AuthMiddleware(s.config.JWTSecret))
	admin.Use(s.adminMiddleware)

	// Admin event routes
	admin.HandleFunc("/events", s.eventHandlers.HandleCreateEvent).Methods("POST", "OPTIONS")
	admin.HandleFunc("/events/{id:[0-9]+}", s.eventHandlers.HandleUpdateEvent).Methods("PUT", "OPTIONS")
	admin.HandleFunc("/events/{id:[0-9]+}", s.eventHandlers.HandleDeleteEvent).Methods("DELETE", "OPTIONS")

	// Admin UMA routes
	admin.HandleFunc("/events/{id:[0-9]+}/uma-invoice", s.eventHandlers.HandleCreateEventUMAInvoice).Methods("POST", "OPTIONS")

	// Admin node balance route
	admin.HandleFunc("/node/balance", s.eventHandlers.HandleGetNodeBalance).Methods("GET", "OPTIONS")

	// Admin payment routes
	admin.HandleFunc("/payments/pending", s.paymentHandlers.HandleGetPendingPayments).Methods("GET", "OPTIONS")
	admin.HandleFunc("/payments/{id:[0-9]+}/retry", s.paymentHandlers.HandleRetryPayment).Methods("POST", "OPTIONS")
}

// Initialize handlers
func (s *Server) initializeHandlers() {
	s.userHandlers = apphandlers.NewUserHandlers(s.userRepo, s.nwcRepo, s.logger, s.config.JWTSecret)
	s.eventHandlers = apphandlers.NewEventHandlers(s.eventRepo, s.paymentRepo, s.ticketRepo, s.umaService, s.umaRepo, s.logger, s.config)
	s.ticketHandlers = apphandlers.NewTicketHandlers(s.ticketRepo, s.eventRepo, s.paymentRepo, s.umaRepo, s.nwcRepo, s.umaService, s.logger, s.config.Domain)
	s.paymentHandlers = apphandlers.NewPaymentHandlers(s.paymentRepo, s.ticketRepo, s.umaService, s.lightsparkClient, s.logger)
	s.lnurlHandlers = apphandlers.NewLnurlHandlers(s.paymentRepo, s.umaService, s.lightsparkClient, s.logger, s.config.Domain, s.config.LightsparkNodeID)
}

// CORS middleware
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return handlers.CORS(
		handlers.AllowedOrigins([]string{
			"http://localhost:3000",  // Local development
			"https://fanmeeting.org", // Production domain
		}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"}),
		handlers.AllowCredentials(),
	)(next)
}

// Logging middleware
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration", duration,
			"user_agent", r.UserAgent(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// Admin middleware
func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUserFromContext(r.Context())
		if user == nil {
			middleware.WriteError(w, http.StatusUnauthorized, "Authentication required")
			return
		}

		// Check if user is admin
		isAdmin := false
		for _, adminEmail := range s.config.AdminEmails {
			if user.Email == adminEmail {
				isAdmin = true
				break
			}
		}

		if !isAdmin {
			middleware.WriteError(w, http.StatusForbidden, "Admin privileges required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		s.logger.Error("Health check failed - database connection error", "error", err)
		middleware.WriteError(w, http.StatusServiceUnavailable, "Database connection failed")
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "tickets-by-uma",
		"version":   "1.0.0",
	})
}

// GetRouter returns the configured router
func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
