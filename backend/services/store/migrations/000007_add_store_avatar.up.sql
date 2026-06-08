-- Store logo/avatar image (object URL in MinIO). Empty until the owner uploads one.
ALTER TABLE stores ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '';
