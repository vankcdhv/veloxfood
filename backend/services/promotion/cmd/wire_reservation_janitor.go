package main

import (
	"context"
	"log/slog"
	"time"

	"project/pkg/app"
	"project/services/promotion/internal/usecase"
)

// startReservationJanitor runs the background sweep that voids orphaned
// RESERVED voucher usages (saga crashed without confirm/release) so
// used_count stops over-counting and vouchers regain their quota.
func startReservationJanitor(a *app.App, applyUC usecase.ApplyUsecase, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		slog.Info("promotion: reservation janitor started", "reserve_ttl", ttl)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				released, err := applyUC.ReleaseExpired(ctx, ttl)
				if err != nil {
					slog.ErrorContext(ctx, "promotion: reservation sweep failed", "err", err)
					continue
				}
				if released > 0 {
					slog.InfoContext(ctx, "promotion: released expired reservations", "orders", released)
				}
			}
		}
	}()

	a.OnShutdown(func() {
		cancel()
		<-done
	})
}
