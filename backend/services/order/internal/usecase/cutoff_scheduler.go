package usecase

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project/services/order/internal/entity"
	"project/services/order/internal/repository"

	"gorm.io/gorm"
)

// CutoffScheduler is a background job that finds READY orders whose service
// date has passed without a shipper being assigned and publishes
// order.cutoff_reached so Delivery can arrange store-self-delivery (BR8).
//
// Multi-instance note: if Order service runs as multiple replicas, each
// instance will independently trigger this scan. A distributed lock
// (e.g. Postgres advisory lock) should be added before going to production
// multi-replica; for MVP a single-instance deployment is assumed.
type CutoffScheduler struct {
	db         *gorm.DB
	orderRepo  repository.OrderRepository
	outboxRepo repository.OutboxRepository
	interval   time.Duration
}

// NewCutoffScheduler constructs the scheduler. interval is typically 1–5 minutes.
func NewCutoffScheduler(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	outboxRepo repository.OutboxRepository,
	interval time.Duration,
) *CutoffScheduler {
	return &CutoffScheduler{
		db:         db,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
		interval:   interval,
	}
}

// Start runs the scheduler loop until ctx is cancelled.
func (s *CutoffScheduler) Start(ctx context.Context) {
	slog.Info("cutoff scheduler started", "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("cutoff scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *CutoffScheduler) tick(ctx context.Context) {
	orders, err := s.orderRepo.FindReadyPastCutoff(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "cutoff scheduler: find orders failed", "err", err)
		return
	}
	for _, o := range orders {
		s.publishCutoffReached(ctx, o)
	}
}

func (s *CutoffScheduler) publishCutoffReached(ctx context.Context, order *entity.Order) {
	payload, _ := json.Marshal(map[string]any{
		"order_id": order.ID,
		"store_id": order.StoreID,
	})

	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "order",
			AggregateID:   order.ID,
			EventType:     "order.cutoff_reached",
			Payload:       payload,
		})
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "cutoff scheduler: publish failed",
			"order_id", order.ID, "err", txErr)
		return
	}
	slog.InfoContext(ctx, "cutoff reached published", "order_id", order.ID, "store_id", order.StoreID)
}
