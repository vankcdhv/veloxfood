package trace

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNew_GeneratesValidUUID(t *testing.T) {
	id := New()
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("New() = %q, want a valid uuid: %v", id, err)
	}
}

func TestWithAndFromContext_RoundTrip(t *testing.T) {
	ctx := WithTraceID(context.Background(), "abc-123")
	if got := FromContext(ctx); got != "abc-123" {
		t.Fatalf("FromContext = %q, want %q", got, "abc-123")
	}
}

func TestWithTraceID_EmptyIsNoOp(t *testing.T) {
	parent := context.Background()
	ctx := WithTraceID(parent, "")
	if ctx != parent {
		t.Fatal("WithTraceID(ctx, \"\") must return parent ctx unchanged")
	}
	if id := FromContext(ctx); id != "" {
		t.Fatalf("FromContext on empty-tagged ctx = %q, want empty", id)
	}
}

func TestFromContext_NilCtx(t *testing.T) {
	if id := FromContext(nil); id != "" { //nolint:staticcheck
		t.Fatalf("FromContext(nil) = %q, want empty", id)
	}
}

func TestEnsureContext_GeneratesIfMissing(t *testing.T) {
	ctx, id := EnsureContext(context.Background())
	if id == "" {
		t.Fatal("EnsureContext on empty ctx must generate an id")
	}
	if FromContext(ctx) != id {
		t.Fatal("EnsureContext must put generated id into returned ctx")
	}
}

func TestEnsureContext_KeepsExisting(t *testing.T) {
	parent := WithTraceID(context.Background(), "keep-me")
	ctx, id := EnsureContext(parent)
	if id != "keep-me" {
		t.Fatalf("EnsureContext id = %q, want keep-me", id)
	}
	if ctx != parent {
		t.Fatal("EnsureContext should return parent ctx unchanged when id exists")
	}
}
