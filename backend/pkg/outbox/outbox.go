// Package outbox implements the transactional outbox pattern for reliable
// event publishing. Usecases call Writer.Insert inside their business
// transaction; a background Dispatcher polls pending rows and publishes them
// to Kafka, marking each row published or failed.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"project/pkg/trace"

	"github.com/google/uuid"
)

// Event is the domain representation passed to Writer.Insert.
// Payload is any JSON-serializable value; it is marshaled to jsonb on INSERT.
type Event struct {
	AggregateType string // e.g. "user", "vendor"
	AggregateID   string // UUID of the aggregate root
	EventType     string // e.g. "user.created", "vendor.requested"
	Payload       any    // marshaled to JSON; must be serializable
}

// Envelope is the JSON object published to Kafka consumers.
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	OccurredAt time.Time       `json:"occurred_at"`
	TraceID    string          `json:"trace_id,omitempty"`
	Data       json.RawMessage `json:"data"`
}

// OutboxRow is the minimal view of outbox_events the dispatcher reads.
type OutboxRow struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       json.RawMessage
	TraceID       string
	Attempts      int
}

// NewEnvelope wraps a raw payload into the standard Kafka envelope.
func NewEnvelope(row *OutboxRow) ([]byte, error) {
	env := Envelope{
		EventID:    row.ID,
		EventType:  row.EventType,
		OccurredAt: time.Now().UTC(),
		TraceID:    row.TraceID,
		Data:       row.Payload,
	}
	b, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope for event %s: %w", row.ID, err)
	}
	return b, nil
}

// TopicFor maps aggregate_type to the Kafka topic name.
// Extend this function as new aggregate types are introduced.
func TopicFor(aggregateType string) string {
	switch aggregateType {
	case "vendor":
		return "vendor.events"
	case "store":
		return "store.events"
	case "payment":
		return "payment.events"
	case "wallet":
		return "wallet.events"
	case "payout":
		return "payout.events"
	default:
		return "user.events"
	}
}

// NewEventID generates a fresh UUID for use as outbox_events.id when callers
// need to pre-assign an ID before INSERT (e.g. for idempotency keys).
func NewEventID() string {
	return uuid.NewString()
}

// TraceIDFromCtx extracts the trace-id from ctx, returning "" if absent.
// Thin shim so usecase packages don't import pkg/trace directly.
func TraceIDFromCtx(ctx context.Context) string {
	return trace.FromContext(ctx)
}
