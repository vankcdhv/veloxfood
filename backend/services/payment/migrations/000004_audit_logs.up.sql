-- Append-only audit trail for privileged actions (who did what to which
-- resource). actor_user_id is a soft reference — users live in user_db.
CREATE TABLE IF NOT EXISTS audit_logs (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_user_id UUID,
  action        VARCHAR(80) NOT NULL,
  target_type   VARCHAR(40),
  target_id     UUID,
  ip            VARCHAR(45),
  user_agent    TEXT,
  trace_id      VARCHAR(80),
  payload       JSONB,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_actor_time ON audit_logs(actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_target     ON audit_logs(target_type, target_id);
