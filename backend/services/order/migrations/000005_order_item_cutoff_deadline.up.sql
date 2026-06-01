-- Snapshot each slot item's actual cutoff deadline (date + cutoff_time − lead)
-- so the cutoff scheduler sweeps a READY order to STORE_DELIVERING only after
-- the slot's real cutoff passes — not merely because the slot date is today.
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS cutoff_deadline TIMESTAMPTZ;
