package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"tickets-by-uma/config"
	"tickets-by-uma/database"
	"tickets-by-uma/server"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("Starting Tickets by UMA backend service")

	// Load configuration
	cfg := config.LoadConfig()
	logger.Info("Configuration loaded", 
		"port", cfg.Port,
		"database_url", maskDatabaseURL(cfg.DatabaseURL),
	)

	// Database connection
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", "error", err)
		}
	}()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", "error", err)
		os.Exit(1)
	}
	logger.Info("Database connection established")

	// Run migrations
	if err := database.RunMigrations(db, logger); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Seed database with initial data
	if err := database.SeedDatabase(db, logger); err != nil {
		logger.Error("Failed to seed database", "error", err)
		os.Exit(1)
	}

	// Create server
	srv := server.NewServer(db, logger, cfg)
	
	// HTTP server setup
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.GetRouter(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Starting HTTP server", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}
	
	logger.Info("Server exited gracefully")
}

// maskDatabaseURL masks sensitive parts of the database URL for logging
func maskDatabaseURL(url string) string {
	// Simple masking - in production, you might want more sophisticated masking
	if len(url) > 20 {
		return url[:10] + "***" + url[len(url)-10:]
	}
	return "***"
}

