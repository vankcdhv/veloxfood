-- processed_events deduplicates incoming Kafka events for the order service.
-- Consumers insert (event_id) before acting; duplicate event_id means already handled.
CREATE TABLE IF NOT EXISTS processed_events (
  event_id   VARCHAR(80) PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
