package trace

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestExtractGRPCMetadata(t *testing.T) {
	md := metadata.Pairs(HeaderHTTPLower, "g-trace")
	ctx := metadata.NewIncomingContext(context.Background(), md)
	if got := ExtractGRPCMetadata(ctx); got != "g-trace" {
		t.Fatalf("ExtractGRPCMetadata = %q, want g-trace", got)
	}
}

func TestExtractGRPCMetadata_Missing(t *testing.T) {
	if got := ExtractGRPCMetadata(context.Background()); got != "" {
		t.Fatalf("ExtractGRPCMetadata on bare ctx = %q, want empty", got)
	}
}

func TestInjectGRPCMetadata_AddsHeader(t *testing.T) {
	ctx := WithTraceID(context.Background(), "out-trace")
	ctx = InjectGRPCMetadata(ctx)

	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatal("expected outgoing metadata to be present")
	}
	got := md.Get(HeaderHTTPLower)
	if len(got) != 1 || got[0] != "out-trace" {
		t.Fatalf("outgoing md = %v, want [out-trace]", got)
	}
}

func TestInjectGRPCMetadata_ReplacesExisting(t *testing.T) {
	base := metadata.Pairs(HeaderHTTPLower, "stale")
	ctx := metadata.NewOutgoingContext(context.Background(), base)
	ctx = WithTraceID(ctx, "fresh")
	ctx = InjectGRPCMetadata(ctx)

	md, _ := metadata.FromOutgoingContext(ctx)
	got := md.Get(HeaderHTTPLower)
	if len(got) != 1 || got[0] != "fresh" {
		t.Fatalf("outgoing md = %v, want exactly [fresh]", got)
	}
}

func TestInjectGRPCMetadata_NoTraceNoOp(t *testing.T) {
	ctx := context.Background()
	if InjectGRPCMetadata(ctx) != ctx {
		t.Fatal("expected no-op when ctx carries no trace id")
	}
}
