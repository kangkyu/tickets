-- migrate:up
-- Change invoice_id columns from VARCHAR(255) to TEXT to accommodate long bolt11 strings

ALTER TABLE payments ALTER COLUMN invoice_id TYPE TEXT;
ALTER TABLE tickets ALTER COLUMN invoice_id TYPE TEXT; 
ALTER TABLE uma_request_invoices ALTER COLUMN invoice_id TYPE TEXT;

-- migrate:down
-- Revert invoice_id columns back to VARCHAR(255)
-- Warning: This may fail if data exists that is longer than 255 characters

ALTER TABLE payments ALTER COLUMN invoice_id TYPE CHARACTER VARYING(255);
ALTER TABLE tickets ALTER COLUMN invoice_id TYPE CHARACTER VARYING(255);
ALTER TABLE uma_request_invoices ALTER COLUMN invoice_id TYPE CHARACTER VARYING(255);