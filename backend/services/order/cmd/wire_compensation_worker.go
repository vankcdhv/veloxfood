package main

import (
	"context"
	"log/slog"
	"time"

	"project/pkg/app"
	"project/services/order/internal/entity"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/repository"
)

const (
	compensationSweepInterval = 30 * time.Second
	compensationMaxAttempts   = 20
	compensationBatchSize     = 50
)

// startCompensationWorker re-drives saga rollbacks that failed in-request
// (rows in pending_compensations). Refund/ReleaseUsage are idempotent by
// order_id, so retrying until success is safe. Rows exceeding the attempt cap
// stay in the table for manual replay and are logged loudly.
func startCompensationWorker(a *app.App, compRepo repository.CompensationRepository,
	paymentClient *grpcclient.PaymentClient, promoClient *grpcclient.PromotionClient) {

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(compensationSweepInterval)
		defer ticker.Stop()
		slog.Info("order: compensation worker started", "interval", compensationSweepInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sweepCompensations(ctx, compRepo, paymentClient, promoClient)
			}
		}
	}()

	a.OnShutdown(func() {
		cancel()
		<-done
	})
}

func sweepCompensations(ctx context.Context, compRepo repository.CompensationRepository,
	paymentClient *grpcclient.PaymentClient, promoClient *grpcclient.PromotionClient) {

	rows, err := compRepo.ListOpen(ctx, compensationMaxAttempts, compensationBatchSize)
	if err != nil {
		slog.ErrorContext(ctx, "order: compensation sweep list failed", "err", err)
		return
	}

	for _, row := range rows {
		var callErr error
		switch row.Action {
		case entity.CompensationRefund:
			if paymentClient == nil {
				callErr = errClientUnavailable
			} else {
				callErr = paymentClient.Refund(ctx, row.OrderID, row.Amount)
			}
		case entity.CompensationReleaseUsage:
			if promoClient == nil {
				callErr = errClientUnavailable
			} else {
				callErr = promoClient.ReleaseUsage(ctx, row.OrderID)
			}
		default:
			slog.ErrorContext(ctx, "order: unknown compensation action, skipping", "id", row.ID, "action", row.Action)
			continue
		}

		if callErr != nil {
			if row.Attempts+1 >= compensationMaxAttempts {
				slog.ErrorContext(ctx, "order: compensation exhausted retries — MANUAL INTERVENTION REQUIRED",
					"id", row.ID, "order_id", row.OrderID, "action", row.Action, "err", callErr)
			}
			if err := compRepo.MarkFailed(ctx, row.ID, callErr.Error()); err != nil {
				slog.ErrorContext(ctx, "order: compensation mark-failed error", "id", row.ID, "err", err)
			}
			continue
		}

		if err := compRepo.MarkDone(ctx, row.ID); err != nil {
			slog.ErrorContext(ctx, "order: compensation mark-done error", "id", row.ID, "err", err)
			continue
		}
		slog.InfoContext(ctx, "order: pending compensation completed",
			"order_id", row.OrderID, "action", row.Action, "attempts", row.Attempts+1)
	}
}

type clientUnavailableError struct{}

func (clientUnavailableError) Error() string { return "grpc client unavailable" }

var errClientUnavailable = clientUnavailableError{}
