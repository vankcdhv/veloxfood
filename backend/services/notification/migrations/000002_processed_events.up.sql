-- processed_events deduplicates incoming Kafka events for the notification service.
CREATE TABLE IF NOT EXISTS processed_events (
  event_id   VARCHAR(80) PRIMARY KEY,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
