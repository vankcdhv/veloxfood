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

	handler := storevent.NewVendorEventHandler(storeUC, quotaUC)

	// Subscribe to vendor.events (vendor lifecycle) using the "store-service" group.
	vendorConsumer := kafka.NewConsumer(brokers, "store-service", "vendor.events")
	vendorCtx, vendorCancel := context.WithCancel(context.Background())

	go func() {
		if err := vendorConsumer.Listen(vendorCtx, handler.HandleKafkaMessage); err != nil {
			slog.Error("store: vendor event consumer stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		vendorCancel()
		if err := vendorConsumer.Close(); err != nil {
			slog.Error("store: vendor event consumer close error", "err", err)
		}
	})

	// Subscribe to order.events (order placed/cancelled → slot quota management)
	// using the same "store-service" group so both topics share a consumer group.
	orderConsumer := kafka.NewConsumer(brokers, "store-service", "order.events")
	orderCtx, orderCancel := context.WithCancel(context.Background())

	go func() {
		if err := orderConsumer.Listen(orderCtx, handler.HandleKafkaMessage); err != nil {
			slog.Error("store: order event consumer stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		orderCancel()
		if err := orderConsumer.Close(); err != nil {
			slog.Error("store: order event consumer close error", "err", err)
		}
	})
}
