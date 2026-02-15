-- migrate:up
ALTER TABLE payments ADD COLUMN preimage text;

-- migrate:down
ALTER TABLE payments DROP COLUMN preimage;
