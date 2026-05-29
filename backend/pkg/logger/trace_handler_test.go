package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"project/pkg/trace"
)

func newJSONLogger(buf *bytes.Buffer) *slog.Logger {
	inner := slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(newTraceHandler(inner))
}

func decodeRecord(t *testing.T, line string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("invalid json log line %q: %v", line, err)
	}
	return m
}

func TestTraceHandler_InjectsTraceIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	log := newJSONLogger(&buf)
	ctx := trace.WithTraceID(context.Background(), "t-aaa")

	log.InfoContext(ctx, "hello")

	rec := decodeRecord(t, strings.TrimSpace(buf.String()))
	if got := rec[trace.LogAttrKey]; got != "t-aaa" {
		t.Fatalf("trace_id = %v, want t-aaa", got)
	}
}

func TestTraceHandler_NoAttrWhenCtxLacksTrace(t *testing.T) {
	var buf bytes.Buffer
	log := newJSONLogger(&buf)
	log.InfoContext(context.Background(), "hello")

	rec := decodeRecord(t, strings.TrimSpace(buf.String()))
	if _, exists := rec[trace.LogAttrKey]; exists {
		t.Fatal("trace_id must not be present when ctx has no id")
	}
}

func TestTraceHandler_WithAttrsKeepsInjection(t *testing.T) {
	// Risk #1 from plan: WithAttrs/WithGroup must keep wrapping.
	var buf bytes.Buffer
	log := newJSONLogger(&buf).With("svc", "x")
	ctx := trace.WithTraceID(context.Background(), "t-bbb")

	log.InfoContext(ctx, "hello after With")

	rec := decodeRecord(t, strings.TrimSpace(buf.String()))
	if got := rec[trace.LogAttrKey]; got != "t-bbb" {
		t.Fatalf("trace_id after .With(...) = %v, want t-bbb", got)
	}
	if got := rec["svc"]; got != "x" {
		t.Fatalf("svc attr = %v, want x", got)
	}
}

func TestTraceHandler_WithGroupKeepsInjection(t *testing.T) {
	// Note: AddAttrs in Handle is subject to the open group at emission time
	// (slog semantics). We document and lock that here — trace_id appears
	// inside the group, not at top level. Business code in this repo doesn't
	// call WithGroup, so the practical query path stays `trace_id`. Tested
	// only to catch regressions where the wrapping is lost entirely.
	var buf bytes.Buffer
	log := newJSONLogger(&buf).WithGroup("g")
	ctx := trace.WithTraceID(context.Background(), "t-ccc")

	log.InfoContext(ctx, "in group")

	rec := decodeRecord(t, strings.TrimSpace(buf.String()))
	group, ok := rec["g"].(map[string]any)
	if !ok {
		t.Fatalf("expected group 'g' in record, got %v", rec)
	}
	if got := group[trace.LogAttrKey]; got != "t-ccc" {
		t.Fatalf("g.trace_id = %v, want t-ccc", got)
	}
}
