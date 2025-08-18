-- migrate:up
-- Remove the unique constraint on invoice_id in payments table
-- Multiple payments can use the same invoice ID (UMA invoices are reusable)
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_invoice_id_key;

-- migrate:down
-- Re-add the unique constraint (if needed for rollback)
ALTER TABLE payments ADD CONSTRAINT payments_invoice_id_key UNIQUE (invoice_id);
