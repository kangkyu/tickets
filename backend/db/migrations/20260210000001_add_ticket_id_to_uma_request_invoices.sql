-- migrate:up
ALTER TABLE uma_request_invoices ADD COLUMN ticket_id integer;
ALTER TABLE uma_request_invoices ALTER COLUMN event_id DROP NOT NULL;
ALTER TABLE uma_request_invoices ADD CONSTRAINT uma_request_invoices_ticket_id_fkey FOREIGN KEY (ticket_id) REFERENCES tickets(id);
CREATE INDEX idx_uma_invoices_ticket_id ON uma_request_invoices USING btree (ticket_id);

-- migrate:down
DROP INDEX IF EXISTS idx_uma_invoices_ticket_id;
ALTER TABLE uma_request_invoices DROP CONSTRAINT IF EXISTS uma_request_invoices_ticket_id_fkey;
ALTER TABLE uma_request_invoices ALTER COLUMN event_id SET NOT NULL;
ALTER TABLE uma_request_invoices DROP COLUMN IF EXISTS ticket_id;
