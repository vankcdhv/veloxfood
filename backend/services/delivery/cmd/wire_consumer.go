package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	pkgkafka "project/pkg/messaging/kafka"
	deliveryevent "project/services/delivery/internal/handler/event"
)

// startOrderEventConsumer subscribes to order.events and handles order lifecycle
// transitions that affect delivery records (ready, cutoff_reached, cancelled).
func startOrderEventConsumer(a *app.App, deps app.Dependencies, handler *deliveryevent.OrderEventHandler) {
	startConsumer(a, deps, "delivery-service", "order.events", handler.HandleKafkaMessage,
		"delivery: order event consumer")
}

// startConsumer is a generic helper that spins up a Kafka consumer goroutine
// and registers shutdown hooks. groupID must be unique per (service, topic) pair.
func startConsumer(
	a *app.App,
	deps app.Dependencies,
	groupID, topic string,
	handler pkgkafka.MessageHandler,
	logPrefix string,
) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — consumer disabled", "topic", topic)
		return
	}

	consumer := pkgkafka.NewConsumer(brokers, groupID, topic)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := consumer.Listen(ctx, handler); err != nil {
			slog.Error(logPrefix+" stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		cancel()
		if err := consumer.Close(); err != nil {
			slog.Error(logPrefix+" close error", "err", err)
		}
	})
}
