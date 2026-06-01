-- Speed up per-target rating summaries and per-item review listings
-- (avg/count GROUP BY target_id, and ListByTarget filtered by target).
CREATE INDEX IF NOT EXISTS idx_reviews_store_target
  ON reviews (store_id, target_type, target_id);

CREATE INDEX IF NOT EXISTS idx_reviews_target
  ON reviews (target_type, target_id);
