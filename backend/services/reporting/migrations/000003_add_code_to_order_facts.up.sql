-- Carry the human-readable order code into the read model so the admin
-- dashboard can show it instead of a raw order id.
ALTER TABLE order_facts ADD COLUMN IF NOT EXISTS code VARCHAR(20);
