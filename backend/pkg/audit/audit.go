// Package audit provides a thin append-only audit-log layer.
// Implementations insert into the audit_logs table using raw SQL (no entity import)
// to avoid import cycles — service entities live in service internal packages.
package audit

import (
	"context"

	"gorm.io/gorm"
)

// Entry is the caller-facing descriptor for one audit event.
type Entry struct {
	// ActorUserID is nil for unauthenticated actions (e.g. register, login-fail).
	ActorUserID *string
	// Action is a dot-separated namespaced verb, e.g. "auth.login_success".
	Action string
	// TargetType identifies the resource category, e.g. "user", "vendor".
	TargetType string
	// TargetID is the resource UUID; nil for actions without a specific target.
	TargetID *string
	// Payload is additional context — will be JSON-marshaled and redacted.
	// Sensitive fields (password, token, code, password_hash) are stripped.
	Payload any
}

// Logger is the narrow interface usecases depend on.
type Logger interface {
	// Record inserts one audit row using the package-level DB connection.
	Record(ctx context.Context, e Entry) error
	// RecordTx inserts inside the caller-supplied *gorm.DB transaction so
	// audit and the business write commit atomically.
	RecordTx(ctx context.Context, tx *gorm.DB, e Entry) error
}

// NoopLogger is a no-op implementation for tests and dependency injection.
type NoopLogger struct{}

func (NoopLogger) Record(_ context.Context, _ Entry) error                { return nil }
func (NoopLogger) RecordTx(_ context.Context, _ *gorm.DB, _ Entry) error { return nil }

// ---- context helpers for IP / User-Agent ----

type ctxMetaKey struct{}

type requestMeta struct {
	IP        string
	UserAgent string
}

// WithRequestMeta stores ip + ua in ctx. Called by the RequestMetadata middleware.
func WithRequestMeta(ctx context.Context, ip, ua string) context.Context {
	return context.WithValue(ctx, ctxMetaKey{}, requestMeta{IP: ip, UserAgent: ua})
}

// IPFromContext returns the client IP stored by WithRequestMeta, or "".
func IPFromContext(ctx context.Context) string {
	if m, ok := ctx.Value(ctxMetaKey{}).(requestMeta); ok {
		return m.IP
	}
	return ""
}

// UserAgentFromContext returns the User-Agent stored by WithRequestMeta, or "".
func UserAgentFromContext(ctx context.Context) string {
	if m, ok := ctx.Value(ctxMetaKey{}).(requestMeta); ok {
		return m.UserAgent
	}
	return ""
}
