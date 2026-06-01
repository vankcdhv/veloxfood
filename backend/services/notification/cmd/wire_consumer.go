package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	pkgkafka "project/pkg/messaging/kafka"
	notifEvent "project/services/notification/internal/handler/event"
)

// startAllConsumers spins up one Kafka consumer goroutine per topic.
// Each consumer uses its own reader (separate group offset tracking) but shares
// the same notification usecase for fan-out.
func startAllConsumers(
	a *app.App,
	deps app.Dependencies,
	orderH *notifEvent.OrderEventHandler,
	deliveryH *notifEvent.DeliveryEventHandler,
	payH *notifEvent.PaymentEventHandler,
	vrH *notifEvent.VendorReviewEventHandler,
) {
	startConsumer(a, deps, "notification-service-order",    "order.events",    orderH.HandleKafkaMessage,           "notification: order consumer")
	startConsumer(a, deps, "notification-service-delivery", "delivery.events",  deliveryH.HandleKafkaMessage,        "notification: delivery consumer")
	startConsumer(a, deps, "notification-service-payment",  "payment.events",   payH.HandlePaymentKafkaMessage,      "notification: payment consumer")
	startConsumer(a, deps, "notification-service-wallet",   "wallet.events",    payH.HandleWalletKafkaMessage,       "notification: wallet consumer")
	startConsumer(a, deps, "notification-service-payout",   "payout.events",    payH.HandlePayoutKafkaMessage,       "notification: payout consumer")
	startConsumer(a, deps, "notification-service-vendor",   "vendor.events",    vrH.HandleVendorKafkaMessage,        "notification: vendor consumer")
	startConsumer(a, deps, "notification-service-review",   "review.events",    vrH.HandleReviewKafkaMessage,        "notification: review consumer")
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
