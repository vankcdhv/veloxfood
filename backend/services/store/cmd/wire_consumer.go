package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	"project/pkg/messaging/kafka"
	storevent "project/services/store/internal/handler/event"
	"project/services/store/internal/usecase"
)

// startVendorEventConsumer subscribes to vendor.events and dispatches messages
// to VendorEventHandler. Runs in a background goroutine; cancelled on shutdown.
func startVendorEventConsumer(
	a *app.App,
	deps app.Dependencies,
	storeUC usecase.StoreUsecase,
	quotaUC usecase.QuotaUsecase,
) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — vendor event consumer disabled")
		return
	}

	consumer := kafka.NewConsumer(brokers, "store-service", "vendor.events")
	handler := storevent.NewVendorEventHandler(storeUC, quotaUC)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := consumer.Listen(ctx, handler.HandleKafkaMessage); err != nil {
			slog.Error("store: vendor event consumer stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		cancel()
		if err := consumer.Close(); err != nil {
			slog.Error("store: vendor event consumer close error", "err", err)
		}
	})
}
