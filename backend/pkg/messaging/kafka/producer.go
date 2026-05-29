package kafka

import (
	"context"
	"log/slog"

	"project/pkg/trace"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}
	slog.Info("kafka producer initialized", "brokers", brokers)
	return &Producer{writer: w}
}

func (p *Producer) Publish(ctx context.Context, topic, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   value,
		Headers: trace.InjectKafkaHeaders(nil, trace.FromContext(ctx)),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
