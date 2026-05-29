package kafka

import (
	"context"
	"log/slog"

	"project/pkg/trace"

	"github.com/segmentio/kafka-go"
)

type MessageHandler func(ctx context.Context, msg kafka.Message) error

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, groupID, topic string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
	})
	slog.Info("kafka consumer initialized", "topic", topic, "group", groupID)
	return &Consumer{reader: r}
}

func (c *Consumer) Listen(ctx context.Context, handler MessageHandler) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
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

		if err := handler(msgCtx, msg); err != nil {
			slog.ErrorContext(msgCtx, "kafka message handler error", "topic", msg.Topic, "error", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
