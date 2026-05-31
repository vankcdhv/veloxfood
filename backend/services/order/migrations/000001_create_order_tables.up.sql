-- Order service schema.
-- Cross-service IDs (customer_id, store_id, location_id, menu_item_id) carry
-- NO FK constraints — each service owns its own data boundary.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Carts ─────────────────────────────────────────────────────────────────────
-- One cart per (customer × store). Items within it accumulate until place-order.
CREATE TABLE IF NOT EXISTS carts (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  customer_id UUID        NOT NULL,
  store_id    UUID        NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (customer_id, store_id)
);
CREATE INDEX IF NOT EXISTS idx_carts_customer ON carts(customer_id);

-- ── Cart items ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS cart_items (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  cart_id      UUID        NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
  menu_item_id UUID        NOT NULL,
  name_snapshot VARCHAR(150) NOT NULL DEFAULT '',
  price_snapshot BIGINT    NOT NULL DEFAULT 0,
  qty          INT         NOT NULL CHECK (qty > 0),
  cutoff_id    UUID,
  date         DATE,
  options_snapshot JSONB   NOT NULL DEFAULT '[]',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (cart_id, menu_item_id)
);
CREATE INDEX IF NOT EXISTS idx_cart_items_cart ON cart_items(cart_id);

-- ── Orders ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS orders (
  id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  code           VARCHAR(20)  NOT NULL UNIQUE,
  customer_id    UUID         NOT NULL,
  store_id       UUID         NOT NULL,
  location_id    UUID,                                        -- NULL for PICKUP
  fulfillment    VARCHAR(10)  NOT NULL CHECK (fulfillment IN ('DELIVERY','PICKUP')),
  status         VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
                 CHECK (status IN ('PENDING','CONFIRMED','PREPARING','READY',
                                   'READY_PICKUP','SHIPPER_ASSIGNED','DELIVERING',
                                   'DELIVERED','COMPLETED','CANCELLED','REJECTED')),
  items_total    BIGINT       NOT NULL DEFAULT 0,
  ship_fee       BIGINT       NOT NULL DEFAULT 0,
  discount       BIGINT       NOT NULL DEFAULT 0,
  grand_total    BIGINT       NOT NULL DEFAULT 0,
  payment_method VARCHAR(10)  NOT NULL CHECK (payment_method IN ('COD','MOMO','WALLET')),
  payment_status VARCHAR(10)  NOT NULL DEFAULT 'UNPAID' CHECK (payment_status IN ('UNPAID','PAID')),
  voucher_codes  JSONB        NOT NULL DEFAULT '[]',
  pickup_pin     VARCHAR(10),                                 -- non-NULL for PICKUP
  placed_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_customer    ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_store       ON orders(store_id);
CREATE INDEX IF NOT EXISTS idx_orders_status      ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_placed_at   ON orders(placed_at DESC);

-- ── Order items ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_items (
  id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id         UUID         NOT NULL REFERENCES orders(id),
  menu_item_id     UUID         NOT NULL,
  name_snapshot    VARCHAR(150) NOT NULL,
  price_snapshot   BIGINT       NOT NULL,
  qty              INT          NOT NULL CHECK (qty > 0),
  cutoff_id        UUID,
  date             DATE,
  options_snapshot JSONB        NOT NULL DEFAULT '[]',
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);

-- ── Order status history ──────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS order_status_history (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id   UUID        NOT NULL REFERENCES orders(id),
  status     VARCHAR(20) NOT NULL,
  changed_by UUID,                                            -- actor user_id; NULL=system
  note       VARCHAR(255),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_osh_order ON order_status_history(order_id);
