package main

import (
	"context"
	"log/slog"

	"project/pkg/app"
	"project/pkg/messaging/kafka"
	promoevent "project/services/promotion/internal/handler/event"
	"project/services/promotion/internal/infrastructure/persistence"
	"project/services/promotion/internal/usecase"
)

// startOrderEventConsumer subscribes to order.events and releases promotion
// usages when an order is cancelled. Group "promotion-service" is dedicated
// so offset tracking is independent of other consumers on the same topic.
func startOrderEventConsumer(a *app.App, deps app.Dependencies, applyUC usecase.ApplyUsecase) {
	brokers := deps.Config.Kafka.Brokers
	if len(brokers) == 0 {
		slog.Warn("kafka brokers not configured — promotion order event consumer disabled")
		return
	}

	processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)
	handler := promoevent.NewOrderEventHandler(deps.DB, applyUC, processedRepo)
	consumer := kafka.NewConsumer(brokers, "promotion-service", "order.events")

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		if err := consumer.Listen(ctx, handler.HandleKafkaMessage); err != nil {
			slog.Error("promotion: order event consumer stopped", "err", err)
		}
	}()

	a.OnShutdown(func() {
		cancel()
		if err := consumer.Close(); err != nil {
			slog.Error("promotion: order event consumer close error", "err", err)
		}
	})
}
