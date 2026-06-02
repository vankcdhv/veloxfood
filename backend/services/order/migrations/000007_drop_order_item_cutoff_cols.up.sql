-- Remove cutoff/slot fields from order_items (scheduler removed, cutoff model dropped).
ALTER TABLE order_items
    DROP COLUMN IF EXISTS cutoff_deadline,
    DROP COLUMN IF EXISTS cutoff_id,
    DROP COLUMN IF EXISTS date;
