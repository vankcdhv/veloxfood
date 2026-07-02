-- Customer favorites: bookmarked stores and menu items.
-- target_id is a soft reference (STORE → stores.id, ITEM → menu_items.id);
-- no FK so a deleted target simply drops out of the joined list queries.
CREATE TABLE IF NOT EXISTS favorites (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL,
  target_type VARCHAR(10) NOT NULL CHECK (target_type IN ('STORE', 'ITEM')),
  target_id   UUID NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_favorites_user_target UNIQUE (user_id, target_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_favorites_user ON favorites (user_id, target_type);
