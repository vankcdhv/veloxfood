package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	pkgkafka "project/pkg/messaging/kafka"
	reviewevent "project/services/review/internal/handler/event"
)

// startOrderEventConsumer subscribes to order.events for eligibility logging.
func startOrderEventConsumer(a *app.App, deps app.Dependencies) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — review order event consumer disabled")
		return
	}

	handler := reviewevent.NewOrderEventHandler()
	startConsumer(a, deps, "review-service-order", "order.events",
		handler.HandleKafkaMessage, "review: order event consumer")
}

// startConsumer is a generic helper that spins up a Kafka consumer goroutine
// and registers shutdown hooks.
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
