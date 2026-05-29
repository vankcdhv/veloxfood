package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"

	"project/pkg/trace"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}

	slog.Info("rabbitmq publisher connected", "url", url)
	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	headers := amqp.Table{}
	trace.InjectAMQPTable(headers, trace.FromContext(ctx))
	return p.channel.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Headers:     headers,
	})
}

func (p *Publisher) DeclareQueue(name string) (amqp.Queue, error) {
	return p.channel.QueueDeclare(name, true, false, false, false, nil)
}

func (p *Publisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return err
	}
	return p.conn.Close()
}
