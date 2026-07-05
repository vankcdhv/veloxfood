package outbox

import (
	"encoding/json"
	"testing"
)

// Every published envelope must carry the schema version, and pre-versioning
// envelopes (no version field) must still decode — consumers treat the empty
// string as version "1".
func TestNewEnvelope_CarriesVersion(t *testing.T) {
	b, err := NewEnvelope(&OutboxRow{
		ID:        "evt-1",
		EventType: "order.placed",
		Payload:   json.RawMessage(`{"order_id":"o1"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	if env.Version != EnvelopeVersion {
		t.Errorf("version: want %q, got %q", EnvelopeVersion, env.Version)
	}
}

func TestEnvelope_PreVersioningDecodes(t *testing.T) {
	legacy := []byte(`{"event_id":"e1","event_type":"order.placed","occurred_at":"2026-01-01T00:00:00Z","data":{}}`)
	var env Envelope
	if err := json.Unmarshal(legacy, &env); err != nil {
		t.Fatalf("legacy envelope must decode: %v", err)
	}
	if env.Version != "" {
		t.Errorf("legacy version: want empty, got %q", env.Version)
	}
}
