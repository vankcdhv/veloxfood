-- Per-day counter backing human-readable order codes (VLX-YYMMDD-NNN).
-- One row per calendar day; the order service upserts and increments it
-- atomically when placing an order.
CREATE TABLE IF NOT EXISTS order_code_seq (
    day DATE PRIMARY KEY,
    seq INTEGER NOT NULL DEFAULT 0
);
