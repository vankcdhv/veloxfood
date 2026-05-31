-- Delivery service schema.
-- Cross-service IDs (order_id, store_id, location_id, customer_id, shipper_id) carry
-- NO FK constraints — each service owns its own data boundary.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Deliveries ────────────────────────────────────────────────────────────────
-- One record per order. order_id is UNIQUE — duplicate order.ready events are safe.
CREATE TABLE IF NOT EXISTS deliveries (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id     UUID        NOT NULL UNIQUE,
  store_id     UUID        NOT NULL,
  location_id  UUID        NOT NULL,
  customer_id  UUID        NOT NULL,
  shipper_id   UUID,                                          -- NULL until claimed
  ship_fee     BIGINT      NOT NULL DEFAULT 0,
  status       VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE'
               CHECK (status IN ('AVAILABLE','CLAIMED','PICKED_UP','DELIVERING',
                                 'DELIVERED','CANCELLED','STORE_DELIVERING')),
  batch_id     UUID,                                          -- NULL until claimed
  claimed_at   TIMESTAMPTZ,
  delivered_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_deliveries_store      ON deliveries(store_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_shipper    ON deliveries(shipper_id);
CREATE INDEX IF NOT EXISTS idx_deliveries_status     ON deliveries(status);

-- ── Delivery batches ──────────────────────────────────────────────────────────
-- One OPEN batch per (shipper, store) at a time. Member count tracked via
-- deliveries.batch_id to avoid a separate counter column.
CREATE TABLE IF NOT EXISTS delivery_batches (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  shipper_id UUID        NOT NULL,
  store_id   UUID        NOT NULL,
  max_size   INT         NOT NULL DEFAULT 5,
  status     VARCHAR(10) NOT NULL DEFAULT 'OPEN'
             CHECK (status IN ('OPEN','CLOSED')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_batches_shipper_store ON delivery_batches(shipper_id, store_id);

-- ── Delivery incidents ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS delivery_incidents (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  delivery_id UUID        NOT NULL,
  order_id    UUID        NOT NULL,
  shipper_id  UUID        NOT NULL,
  type        VARCHAR(50) NOT NULL,
  note        TEXT        NOT NULL DEFAULT '',
  photo_url   TEXT,
  status      VARCHAR(10) NOT NULL DEFAULT 'OPEN'
              CHECK (status IN ('OPEN','RESOLVED')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_incidents_delivery ON delivery_incidents(delivery_id);
