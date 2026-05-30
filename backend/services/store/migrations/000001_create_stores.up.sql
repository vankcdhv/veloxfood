-- Store & Catalog service schema.
-- Soft-delete (deleted_at) on top-level aggregates preserves references from
-- historical orders. Junction/quota tables have no soft-delete.
-- Cross-service IDs (owner_user_id, vendor_id, ref_id) carry NO FK constraints.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Stores ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS stores (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user_id   UUID        NOT NULL,
  vendor_id       UUID        NOT NULL,
  name            VARCHAR(150) NOT NULL,
  business_type   VARCHAR(100),
  address         VARCHAR(255),
  phone           VARCHAR(30),
  sale_status     VARCHAR(20) NOT NULL DEFAULT 'OPEN',
  pickup_enabled  BOOLEAN     NOT NULL DEFAULT FALSE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_stores_owner_user   ON stores(owner_user_id);
-- One active store per vendor (partial unique — allows soft-deleted duplicates).
CREATE UNIQUE INDEX IF NOT EXISTS uq_stores_vendor_active
  ON stores(vendor_id) WHERE deleted_at IS NULL;

-- ── Categories ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS categories (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID         NOT NULL REFERENCES stores(id),
  name       VARCHAR(150) NOT NULL,
  sort_order INT          NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_categories_store ON categories(store_id);

-- ── Menu items ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS menu_items (
  id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id    UUID         NOT NULL REFERENCES stores(id),
  category_id UUID         REFERENCES categories(id),
  name        VARCHAR(150) NOT NULL,
  description TEXT,
  price       BIGINT       NOT NULL,
  image_url   VARCHAR(512),
  status      VARCHAR(8)   NOT NULL DEFAULT 'on',
  tags        VARCHAR(255),
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_menu_items_store    ON menu_items(store_id);
CREATE INDEX IF NOT EXISTS idx_menu_items_category ON menu_items(category_id);

-- ── Menu item slot quotas ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS menu_item_slot_quotas (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  menu_item_id UUID        NOT NULL REFERENCES menu_items(id),
  date         DATE        NOT NULL,
  cutoff_id    UUID        NOT NULL,
  quota        INT         NOT NULL,
  sold_count   INT         NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_slot_quota
  ON menu_item_slot_quotas(menu_item_id, date, cutoff_id);

-- ── Option groups ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS option_groups (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID         NOT NULL REFERENCES stores(id),
  name       VARCHAR(150) NOT NULL,
  min_select INT          NOT NULL DEFAULT 0,
  max_select INT          NOT NULL DEFAULT 1,
  required   BOOLEAN      NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_option_groups_store ON option_groups(store_id);

-- ── Options ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS options (
  id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  option_group_id UUID         NOT NULL REFERENCES option_groups(id),
  name            VARCHAR(150) NOT NULL,
  extra_price     BIGINT       NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_options_group ON options(option_group_id);

-- ── Menu item ↔ option group junction ─────────────────────────────────────────
CREATE TABLE IF NOT EXISTS menu_item_options (
  menu_item_id    UUID        NOT NULL REFERENCES menu_items(id),
  option_group_id UUID        NOT NULL REFERENCES option_groups(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (menu_item_id, option_group_id)
);
CREATE INDEX IF NOT EXISTS idx_mio_menu_item    ON menu_item_options(menu_item_id);
CREATE INDEX IF NOT EXISTS idx_mio_option_group ON menu_item_options(option_group_id);

-- ── Combos ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS combos (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID         NOT NULL REFERENCES stores(id),
  name       VARCHAR(150) NOT NULL,
  price      BIGINT       NOT NULL,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_combos_store ON combos(store_id);

-- ── Combo items ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS combo_items (
  combo_id     UUID        NOT NULL REFERENCES combos(id),
  menu_item_id UUID        NOT NULL REFERENCES menu_items(id),
  quantity     INT         NOT NULL DEFAULT 1,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_combo_items_combo ON combo_items(combo_id);

-- ── Ship fee rules ────────────────────────────────────────────────────────────
-- ref_id points to location-service buildings or rooms — no cross-service FK.
CREATE TABLE IF NOT EXISTS ship_fee_rules (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID        NOT NULL REFERENCES stores(id),
  scope      VARCHAR(10) NOT NULL,
  ref_id     UUID        NOT NULL,
  unit_fee   BIGINT      NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ship_fee_rules_store ON ship_fee_rules(store_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_ship_fee_rule
  ON ship_fee_rules(store_id, scope, ref_id) WHERE deleted_at IS NULL;

-- ── Operating hours ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS operating_hours (
  id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id   UUID        NOT NULL REFERENCES stores(id),
  weekday    SMALLINT    NOT NULL,
  open_time  VARCHAR(5)  NOT NULL,
  close_time VARCHAR(5)  NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_operating_hours_store ON operating_hours(store_id);

-- ── Ship cutoffs ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ship_cutoffs (
  id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id     UUID        NOT NULL REFERENCES stores(id),
  cutoff_time  VARCHAR(5)  NOT NULL,
  lead_minutes INT         NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at   TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ship_cutoffs_store ON ship_cutoffs(store_id);

-- ── Operating hours change requests ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS operating_hours_change_requests (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id    UUID        NOT NULL REFERENCES stores(id),
  payload     JSONB       NOT NULL,
  status      VARCHAR(10) NOT NULL DEFAULT 'pending',
  reviewed_by UUID,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ohcr_store ON operating_hours_change_requests(store_id);
