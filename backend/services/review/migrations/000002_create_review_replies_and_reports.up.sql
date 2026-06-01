CREATE TABLE IF NOT EXISTS review_replies (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  review_id      UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  author_user_id UUID NOT NULL,
  content        TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS review_reports (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  review_id   UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  reported_by UUID NOT NULL,
  reason      TEXT NOT NULL,
  status      VARCHAR(10) NOT NULL DEFAULT 'OPEN'
              CHECK (status IN ('OPEN','RESOLVED')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_review_replies_review_id  ON review_replies(review_id);
CREATE INDEX IF NOT EXISTS idx_review_reports_review_id  ON review_reports(review_id);
CREATE INDEX IF NOT EXISTS idx_review_reports_status     ON review_reports(status);
