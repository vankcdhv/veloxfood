-- Partial index for the reservation janitor: it repeatedly scans for orphaned
-- RESERVED usages older than the TTL, so index exactly that predicate.
CREATE INDEX IF NOT EXISTS idx_promotion_usages_reserved_created
  ON promotion_usages (created_at)
  WHERE status = 'RESERVED';

-- Consumer-side dedupe for Kafka events (at-least-once delivery): handlers
-- insert the envelope event_id here inside their transaction; a duplicate
-- delivery hits the PK and is skipped. Same shape as the other services.
CREATE TABLE IF NOT EXISTS processed_events (
  event_id     UUID PRIMARY KEY,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
