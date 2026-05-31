package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	"project/pkg/messaging/kafka"
	payevent "project/services/payment/internal/handler/event"
)

// startOrderEventConsumer subscribes to order.events and dispatches to OrderEventHandler.
// Runs in a background goroutine; cancelled on shutdown.
func startOrderEventConsumer(a *app.App, deps app.Dependencies, handler *payevent.OrderEventHandler) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — order event consumer disabled")
		return
	}

	// kafka.NewConsumer is single-topic; we use a dedicated consumer per topic.
	consumer := kafka.NewConsumer(brokers, "payment-service", "order.events")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := consumer.Listen(ctx, handler.HandleKafkaMessage); err != nil {
			slog.Error("payment: order event consumer stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		cancel()
		if err := consumer.Close(); err != nil {
			slog.Error("payment: order event consumer close error", "err", err)
		}
	})
}

// startDeliveryEventConsumer subscribes to delivery.events and dispatches to DeliveryEventHandler.
func startDeliveryEventConsumer(a *app.App, deps app.Dependencies, handler *payevent.DeliveryEventHandler) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — delivery event consumer disabled")
		return
	}

	// order.delivered is an order.* event (published to order.events). Use a
	// SEPARATE consumer group so this COD-settlement handler receives every
	// order.events message independently of the placed/cancelled consumer above.
	consumer := kafka.NewConsumer(brokers, "payment-cod-delivered", "order.events")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := consumer.Listen(ctx, handler.HandleKafkaMessage); err != nil {
			slog.Error("payment: delivery event consumer stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		cancel()
		if err := consumer.Close(); err != nil {
			slog.Error("payment: delivery event consumer close error", "err", err)
		}
	})
}
