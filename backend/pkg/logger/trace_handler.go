package logger

import (
	"context"
	"log/slog"

	"project/pkg/trace"
)

// traceHandler wraps an slog.Handler and auto-injects trace_id from ctx into
// every record. Use slog.*Context(ctx, …) call variants for this to fire.
//
// WithAttrs / WithGroup must keep wrapping — otherwise once code does
// logger.With(...) the inner handler returns a bare handler and the trace
// auto-injection silently disappears.
type traceHandler struct{ inner slog.Handler }

func newTraceHandler(inner slog.Handler) slog.Handler {
	return traceHandler{inner: inner}
}

func (h traceHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := trace.FromContext(ctx); id != "" {
		r.AddAttrs(slog.String(trace.LogAttrKey, id))
	}
	return h.inner.Handle(ctx, r)
}

func (h traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return traceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h traceHandler) WithGroup(name string) slog.Handler {
	return traceHandler{inner: h.inner.WithGroup(name)}
}
