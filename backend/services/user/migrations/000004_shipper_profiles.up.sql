-- Shipper onboarding profiles (Admin approval lifecycle).
-- Name/phone live on users; here we keep verification photos (MinIO object keys)
-- and the approval state.
CREATE TABLE IF NOT EXISTS shipper_profiles (
  user_id              UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  id_document_photo_url TEXT NOT NULL,
  portrait_photo_url    TEXT NOT NULL,
  status               VARCHAR(20) NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'approved', 'rejected')),
  approved_by          UUID REFERENCES users(id),
  approved_at          TIMESTAMPTZ,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shipper_status ON shipper_profiles(status);
