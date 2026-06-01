-- Snapshot the human-readable order code onto the delivery so shipper views
-- show VLX-YYMMDD-NNN instead of a raw order id. Populated from order.ready.
ALTER TABLE deliveries ADD COLUMN IF NOT EXISTS order_code VARCHAR(20);
