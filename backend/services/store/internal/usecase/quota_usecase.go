package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"project/services/store/internal/repository"
)

// QuotaUsecase manages slot quota operations for order placement and cancellation.
type QuotaUsecase interface {
	// DecrementSlot atomically increments sold_count. Returns ErrQuotaExceeded when full.
	DecrementSlot(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error

	// RestoreSlot decrements sold_count (order cancellation). Clamps to 0.
	RestoreSlot(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error
}

type quotaUsecase struct {
	catalogRepo repository.CatalogRepository
}

func NewQuotaUsecase(catalogRepo repository.CatalogRepository) QuotaUsecase {
	return &quotaUsecase{catalogRepo: catalogRepo}
}

func (uc *quotaUsecase) DecrementSlot(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error {
	err := uc.catalogRepo.DecrementSlotQuota(ctx, menuItemID, date, cutoffID, qty)
	if errors.Is(err, repository.ErrQuotaExceeded) {
		return ErrQuotaExceeded
	}
	if err != nil {
		return fmt.Errorf("decrement slot quota: %w", err)
	}
	return nil
}

func (uc *quotaUsecase) RestoreSlot(ctx context.Context, menuItemID string, date time.Time, cutoffID string, qty int) error {
	if err := uc.catalogRepo.RestoreSlotQuota(ctx, menuItemID, date, cutoffID, qty); err != nil {
		return fmt.Errorf("restore slot quota: %w", err)
	}
	return nil
}
