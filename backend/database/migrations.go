package database

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// SQL Migration constants
const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);`

const createEventsTable = `
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    price_sats BIGINT NOT NULL CHECK (price_sats > 0),
    stream_url VARCHAR(500),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);`

const createTicketsTable = `
CREATE TABLE IF NOT EXISTS tickets (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id),
    user_id INTEGER NOT NULL REFERENCES users(id),
    ticket_code VARCHAR(255) UNIQUE NOT NULL,
    payment_status VARCHAR(50) DEFAULT 'pending',
    invoice_id VARCHAR(255),
    uma_address VARCHAR(255),
    paid_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);`

const createPaymentsTable = `
CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL REFERENCES tickets(id),
    invoice_id VARCHAR(255) UNIQUE NOT NULL,
    amount_sats BIGINT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    paid_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);`

// Create indexes for better performance
const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_tickets_event_id ON tickets(event_id);
CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON tickets(user_id);
CREATE INDEX IF NOT EXISTS idx_tickets_payment_status ON tickets(payment_status);
CREATE INDEX IF NOT EXISTS idx_payments_ticket_id ON payments(ticket_id);
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
CREATE INDEX IF NOT EXISTS idx_events_is_active ON events(is_active);
`

// Add UMA Request invoice fields to events table
const addUMAFieldsToEvents = `
DO $$
BEGIN
    -- Add uma_request_invoice_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'events' AND column_name = 'uma_request_invoice_id') THEN
        ALTER TABLE events ADD COLUMN uma_request_invoice_id VARCHAR(255);
    END IF;
    
    -- Add uma_request_bolt11 column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'events' AND column_name = 'uma_request_bolt11') THEN
        ALTER TABLE events ADD COLUMN uma_request_bolt11 TEXT;
    END IF;
    
    -- Add uma_request_address column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'events' AND column_name = 'uma_request_address') THEN
        ALTER TABLE events ADD COLUMN uma_request_address VARCHAR(255);
    END IF;
END $$;
`

// Create UMA Request invoices table
const createUMARequestInvoicesTable = `
CREATE TABLE IF NOT EXISTS uma_request_invoices (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    invoice_id VARCHAR(255) UNIQUE NOT NULL,
    payment_hash VARCHAR(255),
    bolt11 TEXT NOT NULL,
    amount_sats BIGINT NOT NULL CHECK (amount_sats > 0),
    status VARCHAR(50) DEFAULT 'pending',
    uma_address VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create index for better performance
CREATE INDEX IF NOT EXISTS idx_uma_invoices_event_id ON uma_request_invoices(event_id);
CREATE INDEX IF NOT EXISTS idx_uma_invoices_status ON uma_request_invoices(status);
`

// RunMigrations executes all database migrations
func RunMigrations(db *sqlx.DB, logger *slog.Logger) error {
	logger.Info("Running database migrations...")

	migrations := []struct {
		name string
		sql  string
	}{
		{"users", createUsersTable},
		{"events", createEventsTable},
		{"tickets", createTicketsTable},
		{"payments", createPaymentsTable},
		{"indexes", createIndexes},
		{"uma_fields", addUMAFieldsToEvents},
		{"uma_invoices", createUMARequestInvoicesTable},
	}

	for _, migration := range migrations {
		logger.Info("Running migration", "table", migration.name)

		if _, err := db.Exec(migration.sql); err != nil {
			logger.Error("Migration failed", "table", migration.name, "error", err)
			return fmt.Errorf("migration %s failed: %w", migration.name, err)
		}

		logger.Info("Migration completed", "table", migration.name)
	}

	logger.Info("All migrations completed successfully")
	return nil
}

// SeedDatabase adds some initial data for testing
func SeedDatabase(db *sqlx.DB, logger *slog.Logger) error {
	logger.Info("Seeding database with initial data...")

	// Check if users already exist
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM users")
	if err != nil {
		return fmt.Errorf("failed to check user count: %w", err)
	}

	if count > 0 {
		logger.Info("Database already seeded, skipping...")
		return nil
	}

	// Insert sample user
	userQuery := `
		INSERT INTO users (email, name, created_at, updated_at) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id`

	var userID int
	err = db.QueryRow(userQuery, "admin@example.com", "Admin User", time.Now(), time.Now()).Scan(&userID)
	if err != nil {
		return fmt.Errorf("failed to create sample user: %w", err)
	}

	// Insert sample event
	eventQuery := `
		INSERT INTO events (title, description, start_time, end_time, capacity, price_sats, stream_url, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	var eventID int
	startTime := time.Now().Add(24 * time.Hour) // Tomorrow
	endTime := startTime.Add(2 * time.Hour)

	err = db.QueryRow(eventQuery,
		"Sample Virtual Event",
		"This is a sample virtual event for testing purposes",
		startTime,
		endTime,
		100,
		1000, // 1000 sats
		"https://stream.example.com/event1",
		true,
		time.Now(),
		time.Now()).Scan(&eventID)

	if err != nil {
		return fmt.Errorf("failed to create sample event: %w", err)
	}

	logger.Info("Database seeded successfully", "user_id", userID, "event_id", eventID)
	return nil
}
