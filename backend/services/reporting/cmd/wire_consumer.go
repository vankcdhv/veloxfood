package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	pkgkafka "project/pkg/messaging/kafka"
	reportevent "project/services/reporting/internal/handler/event"
)

// startOrderEventConsumer subscribes to order.events and updates order_facts + revenue_daily.
func startOrderEventConsumer(a *app.App, deps app.Dependencies, h *reportevent.OrderEventHandler) {
	startConsumer(a, deps, "reporting-service-order", "order.events", h.HandleKafkaMessage,
		"reporting: order event consumer")
}

// startPaymentEventConsumer subscribes to payment.events (payment.captured).
func startPaymentEventConsumer(a *app.App, deps app.Dependencies, h *reportevent.PaymentEventHandler) {
	startConsumer(a, deps, "reporting-service-payment", "payment.events", h.HandlePaymentKafkaMessage,
		"reporting: payment event consumer")
}

// startPayoutEventConsumer subscribes to payout.events (payout.settled → settled=true).
func startPayoutEventConsumer(a *app.App, deps app.Dependencies, h *reportevent.PaymentEventHandler) {
	startConsumer(a, deps, "reporting-service-payout", "payout.events", h.HandlePayoutKafkaMessage,
		"reporting: payout event consumer")
}

// startReviewEventConsumer subscribes to review.events (review.created → store_performance).
func startReviewEventConsumer(a *app.App, deps app.Dependencies, h *reportevent.ReviewVendorEventHandler) {
	startConsumer(a, deps, "reporting-service-review", "review.events", h.HandleReviewKafkaMessage,
		"reporting: review event consumer")
}

// startVendorEventConsumer subscribes to vendor.events (vendor.approved → user_growth_daily).
func startVendorEventConsumer(a *app.App, deps app.Dependencies, h *reportevent.ReviewVendorEventHandler) {
	startConsumer(a, deps, "reporting-service-vendor", "vendor.events", h.HandleVendorKafkaMessage,
		"reporting: vendor event consumer")
}

// startConsumer is a generic helper that spins up a Kafka consumer goroutine
// and registers a shutdown hook. groupID must be unique per (service, topic) pair.
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
