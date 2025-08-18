package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"

	"tickets-by-uma/apphandlers"
	"tickets-by-uma/config"
	"tickets-by-uma/middleware"
	"tickets-by-uma/repositories"
	"tickets-by-uma/services"
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
	umaService      services.UMAService
	router          *mux.Router
	userHandlers    *apphandlers.UserHandlers
	eventHandlers   *apphandlers.EventHandlers
	ticketHandlers  *apphandlers.TicketHandlers
	paymentHandlers *apphandlers.PaymentHandlers
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

	// Initialize UMA service
	s.umaService = services.NewLightsparkUMAService(
		config.LightsparkClientID,
		config.LightsparkClientSecret,
		config.LightsparkNodeID,
		config.LightsparkNodePassword,
		logger,
	)

	// Initialize handlers
	s.initializeHandlers()

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Add CORS middleware to main router (covers all endpoints)
	s.router.Use(s.corsMiddleware)

	// Add logging middleware
	s.router.Use(s.loggingMiddleware)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

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

	// Payment webhook (no auth required)
	api.HandleFunc("/webhooks/payment", s.paymentHandlers.HandlePaymentWebhook).Methods("POST", "OPTIONS")

	// Protected routes (require authentication)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware(s.config.JWTSecret))

	// Protected user routes
	protected.HandleFunc("/users/me", s.userHandlers.HandleGetCurrentUser).Methods("GET", "OPTIONS")
	protected.HandleFunc("/users/{id:[0-9]+}", s.userHandlers.HandleUpdateUser).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/users/{id:[0-9]+}", s.userHandlers.HandleDeleteUser).Methods("DELETE", "OPTIONS")

	// Protected ticket routes
	protected.HandleFunc("/tickets/user/{user_id:[0-9]+}", s.ticketHandlers.HandleGetUserTickets).Methods("GET", "OPTIONS")

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
	admin.HandleFunc("/uma/requests", s.eventHandlers.HandleCreateUMARequest).Methods("POST", "OPTIONS")
	admin.HandleFunc("/events/{id:[0-9]+}/uma-invoice", s.eventHandlers.HandleCreateEventUMAInvoice).Methods("POST", "OPTIONS")

	// Admin payment routes
	admin.HandleFunc("/payments/pending", s.paymentHandlers.HandleGetPendingPayments).Methods("GET", "OPTIONS")
	admin.HandleFunc("/payments/{id:[0-9]+}/retry", s.paymentHandlers.HandleRetryPayment).Methods("POST", "OPTIONS")
}

// Initialize handlers
func (s *Server) initializeHandlers() {
	s.userHandlers = apphandlers.NewUserHandlers(s.userRepo, s.logger, s.config.JWTSecret)
	s.eventHandlers = apphandlers.NewEventHandlers(s.eventRepo, s.paymentRepo, s.umaService, s.umaRepo, s.logger, s.config)
	s.ticketHandlers = apphandlers.NewTicketHandlers(s.ticketRepo, s.eventRepo, s.paymentRepo, s.umaService, s.logger)
	s.paymentHandlers = apphandlers.NewPaymentHandlers(s.paymentRepo, s.ticketRepo, s.umaService, s.logger)
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
