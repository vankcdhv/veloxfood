-- Restore cutoff/slot columns on order_items.
ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS cutoff_id UUID,
    ADD COLUMN IF NOT EXISTS date DATE,
    ADD COLUMN IF NOT EXISTS cutoff_deadline TIMESTAMPTZ;
