package main

import (
	"context"
	"log/slog"
	"time"

	"project/pkg/app"
	"project/services/order/internal/repository"
	"project/services/order/internal/usecase"
)

// startCutoffScheduler launches the BR8 background job that publishes
// order.cutoff_reached for READY orders whose service date has passed
// without a shipper being assigned.
func startCutoffScheduler(
	a *app.App,
	deps app.Dependencies,
	orderRepo repository.OrderRepository,
	outboxRepo repository.OutboxRepository,
) {
	scheduler := usecase.NewCutoffScheduler(
		deps.DB,
		orderRepo,
		outboxRepo,
		2*time.Minute, // poll every 2 minutes
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		scheduler.Start(ctx)
	}()

	a.OnShutdown(func() {
		cancel()
		slog.Info("order: cutoff scheduler stopped")
	})
}
