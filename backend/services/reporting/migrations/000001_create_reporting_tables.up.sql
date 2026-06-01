-- Enable pgcrypto for gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- order_facts: denormalised projection of each order for analytics queries.
CREATE TABLE IF NOT EXISTS order_facts (
  order_id       UUID        PRIMARY KEY,
  store_id       UUID        NOT NULL,
  customer_id    UUID        NOT NULL,
  date           DATE        NOT NULL,
  fulfillment    VARCHAR(10) NOT NULL,
  items_total    BIGINT      NOT NULL DEFAULT 0,
  ship_fee       BIGINT      NOT NULL DEFAULT 0,
  discount       BIGINT      NOT NULL DEFAULT 0,
  grand_total    BIGINT      NOT NULL DEFAULT 0,
  payment_method VARCHAR(10) NOT NULL,
  status         VARCHAR(20) NOT NULL DEFAULT 'PENDING',
  settled        BOOLEAN     NOT NULL DEFAULT FALSE,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_facts_store_date ON order_facts(store_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_order_facts_date       ON order_facts(date DESC);

-- revenue_daily: per-store daily revenue roll-up, updated on each order event.
CREATE TABLE IF NOT EXISTS revenue_daily (
  store_id        UUID  NOT NULL,
  date            DATE  NOT NULL,
  total_food      BIGINT NOT NULL DEFAULT 0,
  total_ship      BIGINT NOT NULL DEFAULT 0,
  orders_count    INT    NOT NULL DEFAULT 0,
  cancelled_count INT    NOT NULL DEFAULT 0,
  PRIMARY KEY (store_id, date)
);

-- user_growth_daily: new registrations per day, keyed by date only (platform-wide).
CREATE TABLE IF NOT EXISTS user_growth_daily (
  date           DATE  PRIMARY KEY,
  new_customers  INT   NOT NULL DEFAULT 0,
  new_vendors    INT   NOT NULL DEFAULT 0
);

-- store_performance: running totals for rating aggregates per store.
CREATE TABLE IF NOT EXISTS store_performance (
  store_id     UUID   PRIMARY KEY,
  rating_sum   BIGINT NOT NULL DEFAULT 0,
  rating_count INT    NOT NULL DEFAULT 0,
  avg_rating   NUMERIC(3,2) NOT NULL DEFAULT 0
);
