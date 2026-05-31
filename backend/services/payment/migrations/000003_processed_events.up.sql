-- processed_events: consumer idempotency — prevents double-processing of Kafka messages.
CREATE TABLE IF NOT EXISTS processed_events (
  event_id     UUID PRIMARY KEY,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
