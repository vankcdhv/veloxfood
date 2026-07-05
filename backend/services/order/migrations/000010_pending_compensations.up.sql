-- Durable saga compensations. When a place-order compensation call
-- (Payment.Refund / Promotion.ReleaseUsage) fails, the intent is persisted
-- here and a background worker retries until it succeeds — a customer can no
-- longer be charged with no refund on record just because the payment service
-- was briefly unreachable during rollback.
CREATE TABLE IF NOT EXISTS pending_compensations (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id   UUID NOT NULL,
  action     VARCHAR(20) NOT NULL CHECK (action IN ('REFUND', 'RELEASE_USAGE')),
  amount     BIGINT NOT NULL DEFAULT 0,
  attempts   INT NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  done_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pending_compensations_open
  ON pending_compensations (created_at)
  WHERE done_at IS NULL;
