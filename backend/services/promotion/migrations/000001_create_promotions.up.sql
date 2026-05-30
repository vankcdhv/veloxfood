-- Promotion service schema.
-- Promotions are scoped per store. usage_limit=NULL means unlimited.
-- Cross-service IDs (store_id, customer_id, order_id) carry NO FK constraints.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Promotions ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS promotions (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id     UUID         NOT NULL,
  code         VARCHAR(50)  NOT NULL,
  type         VARCHAR(20)  NOT NULL,  -- ORDER_DISCOUNT | SHIP_DISCOUNT
  value_kind   VARCHAR(10)  NOT NULL,  -- PERCENT | AMOUNT
  value        BIGINT       NOT NULL,
  min_order    BIGINT       NOT NULL DEFAULT 0,
  max_discount BIGINT,                 -- cap when value_kind=PERCENT; NULL=no cap
  starts_at    TIMESTAMPTZ  NOT NULL,
  ends_at      TIMESTAMPTZ  NOT NULL,
  usage_limit  INT,                    -- NULL = unlimited
  used_count   INT          NOT NULL DEFAULT 0,
  status       VARCHAR(10)  NOT NULL DEFAULT 'ACTIVE', -- ACTIVE | INACTIVE
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at   TIMESTAMPTZ,
  CONSTRAINT chk_promotions_timeframe CHECK (starts_at <= ends_at)
);

-- One active code per store (allows soft-deleted duplicates).
CREATE UNIQUE INDEX IF NOT EXISTS uq_promotions_store_code
  ON promotions(store_id, code) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_promotions_store    ON promotions(store_id);
CREATE INDEX IF NOT EXISTS idx_promotions_status   ON promotions(status);

-- ── Promotion usages ─────────────────────────────────────────────────────────
-- Tracks reservation/confirmation/void lifecycle per order+promotion pair.
CREATE TABLE IF NOT EXISTS promotion_usages (
  id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  promotion_id   UUID        NOT NULL REFERENCES promotions(id),
  order_id       UUID        NOT NULL,
  customer_id    UUID        NOT NULL,
  code           VARCHAR(50) NOT NULL,
  type           VARCHAR(20) NOT NULL,  -- ORDER_DISCOUNT | SHIP_DISCOUNT (lets re-apply rebuild per-type breakdown)
  applied_amount BIGINT      NOT NULL,
  status         VARCHAR(10) NOT NULL DEFAULT 'RESERVED', -- RESERVED | CONFIRMED | VOIDED
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotency: one usage row per (order, promotion).
CREATE UNIQUE INDEX IF NOT EXISTS uq_promotion_usage
  ON promotion_usages(order_id, promotion_id);

-- Efficient lookup of active reservations for a promotion.
CREATE INDEX IF NOT EXISTS idx_promotion_usages_promo_status
  ON promotion_usages(promotion_id, status);

CREATE INDEX IF NOT EXISTS idx_promotion_usages_order
  ON promotion_usages(order_id);
