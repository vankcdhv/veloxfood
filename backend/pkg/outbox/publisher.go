package outbox

import (
	"context"

	"project/pkg/messaging/kafka"
	"project/pkg/trace"
)

// Publisher sends a serialized event envelope to the message broker.
// The interface lives in pkg/outbox to keep the Dispatcher import-free of
// concrete broker packages (prevents circular deps if kafka ever imports outbox).
type Publisher interface {
	// Publish sends payload to topic, keyed by key.
	// traceID is injected into the broker message headers via the context.
	Publish(ctx context.Context, topic, key string, payload []byte, traceID string) error
}

// kafkaPublisher adapts pkg/messaging/kafka.Producer to the Publisher interface.
type kafkaPublisher struct {
	producer *kafka.Producer
}

// NewKafkaPublisher wraps a Kafka producer as an outbox Publisher.
func NewKafkaPublisher(producer *kafka.Producer) Publisher {
	return &kafkaPublisher{producer: producer}
}

// Publish injects traceID into ctx so that Producer.Publish attaches the
// x-trace-id Kafka header automatically via pkg/trace.InjectKafkaHeaders.
func (p *kafkaPublisher) Publish(ctx context.Context, topic, key string, payload []byte, traceID string) error {
	ctx = trace.WithTraceID(ctx, traceID)
	return p.producer.Publish(ctx, topic, key, payload)
}
