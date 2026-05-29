// Package trace carries a per-request trace-id through context.Context and
// across transport boundaries (HTTP, gRPC, Kafka, RabbitMQ). The logger
// auto-injects the id into every *Context slog call so business code never
// has to thread it explicitly.
package trace

import (
	"context"

	"github.com/google/uuid"
)

const (
	// HeaderHTTP is the canonical mixed-case HTTP header name.
	HeaderHTTP = "X-Trace-Id"
	// HeaderHTTPLower is the gRPC metadata / lowercased form.
	HeaderHTTPLower = "x-trace-id"
	// KafkaHeader is the kafka message header key.
	KafkaHeader = "x-trace-id"
	// AMQPHeader is the amqp table key.
	AMQPHeader = "x-trace-id"
	// LogAttrKey is the slog attribute key used by the trace handler.
	LogAttrKey = "trace_id"
)

type ctxKey struct{}

// New returns a fresh UUIDv4 string.
func New() string {
	return uuid.NewString()
}

// WithTraceID returns a derived context carrying id. Empty id is a no-op so
// callers can chain Extract → WithTraceID without a nil check.
func WithTraceID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the trace id from ctx, or "" if none.
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// EnsureContext returns ctx with a trace id, generating one if absent.
// Use at every inbound entry point.
func EnsureContext(ctx context.Context) (context.Context, string) {
	if id := FromContext(ctx); id != "" {
		return ctx, id
	}
	id := New()
	return WithTraceID(ctx, id), id
}
