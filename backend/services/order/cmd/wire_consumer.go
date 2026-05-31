package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	pkgkafka "project/pkg/messaging/kafka"
	orderevent "project/services/order/internal/handler/event"
)

// startPaymentEventConsumer subscribes to payment.events and updates order payment_status.
func startPaymentEventConsumer(a *app.App, deps app.Dependencies, handler *orderevent.PaymentEventHandler) {
	startConsumer(a, deps, "payment-events-order", "payment.events", handler.HandleKafkaMessage,
		"order: payment event consumer")
}

// startDeliveryEventConsumer subscribes to delivery.events and syncs order delivery status.
func startDeliveryEventConsumer(a *app.App, deps app.Dependencies, handler *orderevent.DeliveryEventHandler) {
	startConsumer(a, deps, "delivery-events-order", "delivery.events", handler.HandleKafkaMessage,
		"order: delivery event consumer")
}

// startStoreEventConsumer subscribes to store.events (store status changes).
func startStoreEventConsumer(a *app.App, deps app.Dependencies, handler *orderevent.StoreEventHandler) {
	startConsumer(a, deps, "store-events-order", "store.events", handler.HandleKafkaMessage,
		"order: store event consumer")
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
