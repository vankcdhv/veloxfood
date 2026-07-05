package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/trace"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Delivery contract: at-least-once. Offsets are committed only after the
// handler succeeds or the message has been parked on the dead-letter topic,
// so a transient failure (DB down, dependency timeout) can never silently
// drop a business event. Handlers must stay idempotent (processed_events /
// natural keys) because redelivery is possible.
type Consumer struct {
	reader      *kafka.Reader
	dlqWriter   *kafka.Writer
	dlqTopic    string
	maxRetries  int
	baseBackoff time.Duration
}

type ConsumerOption func(*Consumer)

// WithMaxRetries overrides the in-process retry count before a message is
// sent to the dead-letter topic (default 3).
func WithMaxRetries(n int) ConsumerOption {
	return func(c *Consumer) { c.maxRetries = n }
}

// WithoutDLQ disables the dead-letter topic; exhausted messages are then
// logged and skipped (previous behaviour).
func WithoutDLQ() ConsumerOption {
	return func(c *Consumer) {
		if c.dlqWriter != nil {
			_ = c.dlqWriter.Close()
		}
		c.dlqWriter = nil
	}
}

func NewConsumer(brokers []string, groupID, topic string, opts ...ConsumerOption) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
	})
	c := &Consumer{
		reader:      r,
		dlqTopic:    topic + ".dlq",
		maxRetries:  3,
		baseBackoff: 500 * time.Millisecond,
		dlqWriter: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Balancer:               &kafka.LeastBytes{},
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	slog.Info("kafka consumer initialized", "topic", topic, "group", groupID)
	return c
}

func (c *Consumer) Listen(ctx context.Context, handler MessageHandler) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.ErrorContext(ctx, "kafka read error", "error", err)
			continue
		}

		id := trace.ExtractKafkaHeaders(msg.Headers)
		if id == "" {
			id = trace.New()
		}
		msgCtx := trace.WithTraceID(ctx, id)

		// One consumer span per message; the legacy trace-id rides along as an
		// attribute so logs and Jaeger cross-reference. No-op when tracing is
		// disabled (default TracerProvider).
		msgCtx, span := otel.Tracer("kafka-consumer").Start(msgCtx, "consume "+msg.Topic,
			oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
			oteltrace.WithAttributes(
				attribute.String("messaging.system", "kafka"),
				attribute.String("messaging.destination.name", msg.Topic),
				attribute.String("app.trace_id", id),
			),
		)

		if err := c.process(msgCtx, msg, handler); err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.End()
			// Could not handle nor park the message. Committing here would
			// lose it, so back off and re-deliver (blocks this partition —
			// ordering is preserved on purpose).
			slog.ErrorContext(msgCtx, "kafka message unrecoverable this round, will redeliver",
				"topic", msg.Topic, "offset", msg.Offset, "error", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
			continue
		}

		span.End()

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.ErrorContext(msgCtx, "kafka commit failed", "topic", msg.Topic, "offset", msg.Offset, "error", err)
		}
	}
}

// process runs the handler with bounded retries, then falls back to the DLQ.
// A nil return means the offset is safe to commit.
func (c *Consumer) process(ctx context.Context, msg kafka.Message, handler MessageHandler) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.baseBackoff << (attempt - 1)):
			}
		}
		if lastErr = handler(ctx, msg); lastErr == nil {
			return nil
		}
		slog.ErrorContext(ctx, "kafka message handler error",
			"topic", msg.Topic, "offset", msg.Offset, "attempt", attempt+1, "error", lastErr)
	}

	if c.dlqWriter == nil {
		slog.ErrorContext(ctx, "kafka message dropped after retries (dlq disabled)",
			"topic", msg.Topic, "offset", msg.Offset, "error", lastErr)
		return nil
	}
	if err := c.sendToDLQ(ctx, msg, lastErr); err != nil {
		return fmt.Errorf("dlq publish: %w", err)
	}
	slog.WarnContext(ctx, "kafka message parked on dead-letter topic",
		"topic", msg.Topic, "dlq", c.dlqTopic, "offset", msg.Offset, "error", lastErr)
	return nil
}

func (c *Consumer) sendToDLQ(ctx context.Context, msg kafka.Message, cause error) error {
	headers := append([]kafka.Header{}, msg.Headers...)
	headers = append(headers,
		kafka.Header{Key: "x-original-topic", Value: []byte(msg.Topic)},
		kafka.Header{Key: "x-original-partition", Value: []byte(fmt.Sprintf("%d", msg.Partition))},
		kafka.Header{Key: "x-original-offset", Value: []byte(fmt.Sprintf("%d", msg.Offset))},
		kafka.Header{Key: "x-error", Value: []byte(cause.Error())},
		kafka.Header{Key: "x-failed-at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
	)
	return c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Topic:   c.dlqTopic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headers,
	})
}

func (c *Consumer) Close() error {
	if c.dlqWriter != nil {
		_ = c.dlqWriter.Close()
	}
	return c.reader.Close()
}
