package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/trace"

	"gorm.io/gorm"
)

// sensitiveKeys are stripped from payload maps before JSON serialization.
// Prevents accidental logging of secrets.
var sensitiveKeys = map[string]struct{}{
	"password":      {},
	"password_hash": {},
	"token":         {},
	"code":          {},
	"secret":        {},
	"otp":           {},
}

// gormLogger inserts audit rows using raw table name to avoid importing service entities.
type gormLogger struct {
	db *gorm.DB
}

// NewGormLogger returns a Logger backed by the given *gorm.DB.
func NewGormLogger(db *gorm.DB) Logger {
	return &gormLogger{db: db}
}

// Record inserts an audit_log row using the package-level DB connection.
func (l *gormLogger) Record(ctx context.Context, e Entry) error {
	return l.insert(ctx, l.db.WithContext(ctx), e)
}

// RecordTx inserts inside the caller-supplied transaction.
func (l *gormLogger) RecordTx(ctx context.Context, tx *gorm.DB, e Entry) error {
	return l.insert(ctx, tx.WithContext(ctx), e)
}

func (l *gormLogger) insert(ctx context.Context, db *gorm.DB, e Entry) error {
	ip := strPtrOrNil(IPFromContext(ctx))
	ua := strPtrOrNil(UserAgentFromContext(ctx))
	traceID := strPtrOrNil(trace.FromContext(ctx))

	var payloadJSON *json.RawMessage
	if e.Payload != nil {
		redacted := redactSensitive(e.Payload)
		b, err := json.Marshal(redacted)
		if err != nil {
			slog.WarnContext(ctx, "audit: marshal payload failed", "action", e.Action, "err", err)
		} else {
			raw := json.RawMessage(b)
			payloadJSON = &raw
		}
	}

	targetType := strPtrOrNil(e.TargetType)

	row := map[string]any{
		"actor_user_id": e.ActorUserID,
		"action":        e.Action,
		"target_type":   targetType,
		"target_id":     e.TargetID,
		"ip":            ip,
		"user_agent":    ua,
		"trace_id":      traceID,
		"payload":       payloadJSON,
	}

	if err := db.Table("audit_logs").Create(row).Error; err != nil {
		slog.WarnContext(ctx, "audit: insert failed", "action", e.Action, "err", err)
		return err
	}
	return nil
}

// redactSensitive returns a copy of payload with sensitive keys removed.
// Handles map[string]any and map[string]string; returns other types unchanged.
func redactSensitive(payload any) any {
	switch v := payload.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, val := range v {
			if _, sensitive := sensitiveKeys[k]; !sensitive {
				out[k] = val
			}
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, val := range v {
			if _, sensitive := sensitiveKeys[k]; !sensitive {
				out[k] = val
			}
		}
		return out
	default:
		return payload
	}
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
