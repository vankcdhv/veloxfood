-- notifications stores in-app, push, and email notification records per user.
CREATE TABLE IF NOT EXISTS notifications (
  id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID         NOT NULL,
  type       VARCHAR(60)  NOT NULL,
  title      VARCHAR(255) NOT NULL,
  body       TEXT         NOT NULL,
  data       JSONB        NOT NULL DEFAULT '{}',
  channel    VARCHAR(20)  NOT NULL DEFAULT 'in_app' CHECK (channel IN ('push','email','in_app')),
  read_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id      ON notifications (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user_unread  ON notifications (user_id, read_at) WHERE read_at IS NULL;

-- device_tokens stores FCM push tokens per user device.
CREATE TABLE IF NOT EXISTS device_tokens (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID        NOT NULL,
  fcm_token   VARCHAR(255) NOT NULL UNIQUE,
  platform    VARCHAR(20) NOT NULL DEFAULT 'web' CHECK (platform IN ('ios','android','web')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens (user_id);
