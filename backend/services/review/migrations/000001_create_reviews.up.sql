CREATE TABLE IF NOT EXISTS reviews (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id    UUID NOT NULL,
  customer_id UUID NOT NULL,
  store_id    UUID NOT NULL,
  target_type VARCHAR(10) NOT NULL CHECK (target_type IN ('STORE','ITEM','SHIPPER')),
  target_id   UUID NOT NULL,
  rating      SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment     TEXT NOT NULL DEFAULT '',
  photo_urls  JSONB NOT NULL DEFAULT '[]',
  status      VARCHAR(10) NOT NULL DEFAULT 'VISIBLE'
              CHECK (status IN ('VISIBLE','HIDDEN')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (order_id, target_type, target_id)
);

CREATE INDEX IF NOT EXISTS idx_reviews_store_id   ON reviews(store_id);
CREATE INDEX IF NOT EXISTS idx_reviews_customer_id ON reviews(customer_id);
CREATE INDEX IF NOT EXISTS idx_reviews_status      ON reviews(status);
