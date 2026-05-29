package trace

import (
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestInjectExtractKafkaHeaders(t *testing.T) {
	got := InjectKafkaHeaders(nil, "k-trace")
	if extracted := ExtractKafkaHeaders(got); extracted != "k-trace" {
		t.Fatalf("ExtractKafkaHeaders = %q, want k-trace", extracted)
	}
}

func TestInjectKafkaHeaders_ReplacesExisting(t *testing.T) {
	in := []kafka.Header{
		{Key: "x-other", Value: []byte("keep")},
		{Key: KafkaHeader, Value: []byte("stale")},
	}
	out := InjectKafkaHeaders(in, "fresh")

	count := 0
	for _, h := range out {
		if h.Key == KafkaHeader {
			count++
			if string(h.Value) != "fresh" {
				t.Fatalf("trace header value = %q, want fresh", h.Value)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 trace header, got %d", count)
	}
	// other headers preserved
	found := false
	for _, h := range out {
		if h.Key == "x-other" && string(h.Value) == "keep" {
			found = true
		}
	}
	if !found {
		t.Fatal("non-trace headers must be preserved")
	}
}

func TestInjectKafkaHeaders_EmptyNoOp(t *testing.T) {
	in := []kafka.Header{{Key: "a", Value: []byte("b")}}
	out := InjectKafkaHeaders(in, "")
	if len(out) != 1 || out[0].Key != "a" {
		t.Fatalf("expected input unchanged, got %v", out)
	}
}
