-- migrate:up
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

CREATE INDEX IF NOT EXISTS idx_uma_invoices_event_id ON uma_request_invoices(event_id);
CREATE INDEX IF NOT EXISTS idx_uma_invoices_status ON uma_request_invoices(status);

-- migrate:down
DROP TABLE IF EXISTS uma_request_invoices;

