package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"

	"project/pkg/trace"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(ctx context.Context, delivery amqp.Delivery) error

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewConsumer(url string) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	slog.Info("rabbitmq consumer connected")
	return &Consumer{conn: conn, channel: ch}, nil
}

func (c *Consumer) Listen(ctx context.Context, queueName string, handler MessageHandler) error {
	msgs, err := c.channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			id := trace.ExtractAMQPTable(msg.Headers)
			if id == "" {
				id = trace.New()
			}
			msgCtx := trace.WithTraceID(ctx, id)

			if err := handler(msgCtx, msg); err != nil {
				slog.ErrorContext(msgCtx, "rabbitmq handler error", "queue", queueName, "error", err)
				msg.Nack(false, true)
				continue
			}
			msg.Ack(false)
		}
	}
}

func (c *Consumer) Close() error {
	if err := c.channel.Close(); err != nil {
		return err
	}
	return c.conn.Close()
}
