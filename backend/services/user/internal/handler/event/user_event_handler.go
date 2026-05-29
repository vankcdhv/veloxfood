package event

import (
	"context"
	"encoding/json"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/segmentio/kafka-go"

	"project/services/user/internal/usecase"
)

type UserEventHandler struct {
	userUC usecase.UserUsecase
}

func NewUserEventHandler(userUC usecase.UserUsecase) *UserEventHandler {
	return &UserEventHandler{userUC: userUC}
}

type UserCreatedEvent struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// HandleKafkaMessage processes Kafka events related to users.
func (h *UserEventHandler) HandleKafkaMessage(ctx context.Context, msg kafka.Message) error {
	slog.InfoContext(ctx, "kafka event received", "topic", msg.Topic, "key", string(msg.Key))

	var event UserCreatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal kafka message", "error", err)
		return err
	}

	slog.InfoContext(ctx, "processed kafka user event", "user_id", event.UserID)
	return nil
}

// HandleRabbitMQMessage processes RabbitMQ task messages.
func (h *UserEventHandler) HandleRabbitMQMessage(ctx context.Context, delivery amqp.Delivery) error {
	slog.InfoContext(ctx, "rabbitmq message received", "routing_key", delivery.RoutingKey)

	var event UserCreatedEvent
	if err := json.Unmarshal(delivery.Body, &event); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal rabbitmq message", "error", err)
		return err
	}

	slog.InfoContext(ctx, "processed rabbitmq user task", "user_id", event.UserID)
	return nil
}
