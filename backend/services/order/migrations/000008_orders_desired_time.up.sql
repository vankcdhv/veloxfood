-- Customer's desired receive time (ASAP when null).
ALTER TABLE orders ADD COLUMN IF NOT EXISTS desired_time TIMESTAMPTZ;
