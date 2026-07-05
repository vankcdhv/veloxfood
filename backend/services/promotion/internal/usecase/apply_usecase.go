package usecase

import (
	"context"
	"log/slog"
	"time"

	"project/services/promotion/internal/entity"
	"project/services/promotion/internal/repository"

	"gorm.io/gorm"
)

// ApplyResult is the outcome of a successful ApplyPromotion call.
type ApplyResult struct {
	Success      bool
	ItemDiscount int64
	ShipDiscount int64
	Applied      []AppliedCode
	ErrorReason  string
}

// AppliedCode carries the per-code discount breakdown returned to callers.
type AppliedCode struct {
	Code   string
	Type   string
	Amount int64
}

// ApplyRequest is the input for reserving promotions on an order.
type ApplyRequest struct {
	OrderID    string
	StoreID    string
	CustomerID string
	Codes      []string
	Subtotal   int64
	ItemCount  int32
}

// ApplyUsecase handles the saga-participant gRPC operations.
type ApplyUsecase interface {
	// ApplyPromotion reserves one or more promotion codes for an order.
	// Idempotent: re-sending the same order_id returns the existing breakdown.
	ApplyPromotion(ctx context.Context, req ApplyRequest) (*ApplyResult, error)

	// QuotePromotion computes the exact breakdown ApplyPromotion would grant,
	// without reserving quota. Order quotes the price before opening a saga.
	QuotePromotion(ctx context.Context, req ApplyRequest) (*ApplyResult, error)

	// ConfirmUsage transitions RESERVED usages for an order to CONFIRMED.
	ConfirmUsage(ctx context.Context, orderID string) error

	// ReleaseUsage voids RESERVED usages for an order and decrements used_count.
	ReleaseUsage(ctx context.Context, orderID string) error

	// ReleaseUsageInTx is ReleaseUsage running inside a caller-owned
	// transaction — used by event handlers that must commit the release
	// atomically with their processed_events dedupe row.
	ReleaseUsageInTx(ctx context.Context, tx *gorm.DB, orderID string) error

	// ReleaseExpired voids all RESERVED usages older than ttl (orphaned
	// reservations whose saga never confirmed nor released) and returns how
	// many orders were cleaned.
	ReleaseExpired(ctx context.Context, ttl time.Duration) (int, error)
}

type applyUsecase struct {
	db        *gorm.DB
	promoRepo repository.PromotionRepository
	usageRepo repository.PromotionUsageRepository
}

// NewApplyUsecase constructs ApplyUsecase.
func NewApplyUsecase(
	db *gorm.DB,
	promoRepo repository.PromotionRepository,
	usageRepo repository.PromotionUsageRepository,
) ApplyUsecase {
	return &applyUsecase{
		db:        db,
		promoRepo: promoRepo,
		usageRepo: usageRepo,
	}
}

// ApplyPromotion runs in a single serialised transaction per order.
// FOR UPDATE locks on each promotion row prevent concurrent over-redemption when
// two requests race on the same voucher with usage_limit=1.
func (uc *applyUsecase) ApplyPromotion(ctx context.Context, req ApplyRequest) (*ApplyResult, error) {
	// Idempotency check: if RESERVED usages already exist for this order, return
	// the existing breakdown without touching the DB further.
	existing, err := uc.usageRepo.ListByOrderID(ctx, req.OrderID)
	if err != nil {
		return failResult("internal error"), nil
	}
	if len(existing) > 0 {
		return buildResultFromExisting(existing), nil
	}

	var result *ApplyResult
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var appliedCodes []AppliedCode
		var itemDiscount, shipDiscount int64

		// Track types already granted in this request to enforce ≤1 per type.
		grantedTypes := map[string]bool{}

		for _, code := range req.Codes {
			p, err := uc.promoRepo.GetByStoreAndCodeForUpdate(ctx, tx, req.StoreID, code)
			if err != nil {
				result = failResult("promotion not found: " + code)
				return errRollback
			}

			if reason := validatePromotion(p, req.StoreID, req.Subtotal, now); reason != "" {
				result = failResult(reason)
				return errRollback
			}

			// Enforce at most one ORDER_DISCOUNT and one SHIP_DISCOUNT per order.
			if grantedTypes[p.Type] {
				result = failResult("cannot apply two " + p.Type + " promotions to one order")
				return errRollback
			}
			grantedTypes[p.Type] = true

			amount := computeDiscount(p, req.Subtotal)

			// Insert usage row (UNIQUE(order_id, promotion_id) prevents double-insert).
			usage := &entity.PromotionUsage{
				PromotionID:   p.ID,
				OrderID:       req.OrderID,
				CustomerID:    req.CustomerID,
				Code:          code,
				Type:          p.Type,
				AppliedAmount: amount,
				Status:        "RESERVED",
			}
			if err := uc.usageRepo.Create(ctx, tx, usage); err != nil {
				result = failResult("failed to reserve promotion")
				return errRollback
			}

			if err := uc.promoRepo.IncrementUsedCount(ctx, tx, p.ID); err != nil {
				result = failResult("failed to reserve promotion")
				return errRollback
			}

			appliedCodes = append(appliedCodes, AppliedCode{Code: code, Type: p.Type, Amount: amount})
			switch p.Type {
			case "ORDER_DISCOUNT":
				itemDiscount += amount
			case "SHIP_DISCOUNT":
				shipDiscount += amount
			}
		}

		result = &ApplyResult{
			Success:      true,
			ItemDiscount: itemDiscount,
			ShipDiscount: shipDiscount,
			Applied:      appliedCodes,
		}
		return nil
	})

	if txErr != nil && txErr != errRollback {
		slog.ErrorContext(ctx, "promotion: ApplyPromotion tx error", "order_id", req.OrderID, "err", txErr)
		return failResult("internal error"), nil
	}
	if result == nil {
		return failResult("internal error"), nil
	}
	return result, nil
}

// QuotePromotion is the dry-run twin of ApplyPromotion: same validation, same
// per-type cap, same discount math — but no usage rows, no used_count bump.
// Sharing validatePromotion/computeDiscount keeps quote == apply by
// construction; a mismatch at apply time means state changed in between
// (e.g. the voucher ran out) and the saga branch must abort.
func (uc *applyUsecase) QuotePromotion(ctx context.Context, req ApplyRequest) (*ApplyResult, error) {
	now := time.Now()
	var applied []AppliedCode
	var itemDiscount, shipDiscount int64
	grantedTypes := map[string]bool{}

	for _, code := range req.Codes {
		p, err := uc.promoRepo.GetByStoreAndCodeForUpdate(ctx, uc.db.WithContext(ctx), req.StoreID, code)
		if err != nil {
			return failResult("promotion not found: " + code), nil
		}
		if reason := validatePromotion(p, req.StoreID, req.Subtotal, now); reason != "" {
			return failResult(reason), nil
		}
		if grantedTypes[p.Type] {
			return failResult("cannot apply two " + p.Type + " promotions to one order"), nil
		}
		grantedTypes[p.Type] = true

		amount := computeDiscount(p, req.Subtotal)
		applied = append(applied, AppliedCode{Code: code, Type: p.Type, Amount: amount})
		switch p.Type {
		case "ORDER_DISCOUNT":
			itemDiscount += amount
		case "SHIP_DISCOUNT":
			shipDiscount += amount
		}
	}

	return &ApplyResult{
		Success:      true,
		ItemDiscount: itemDiscount,
		ShipDiscount: shipDiscount,
		Applied:      applied,
	}, nil
}

// ConfirmUsage transitions RESERVED → CONFIRMED. Idempotent.
func (uc *applyUsecase) ConfirmUsage(ctx context.Context, orderID string) error {
	if err := uc.usageRepo.ConfirmByOrderID(ctx, orderID); err != nil {
		slog.ErrorContext(ctx, "promotion: ConfirmUsage failed", "order_id", orderID, "err", err)
		return err
	}
	slog.InfoContext(ctx, "promotion: usages confirmed", "order_id", orderID)
	return nil
}

// ReleaseUsage voids RESERVED usages and decrements used_count for each voided
// row. Idempotent — no-op when no RESERVED rows exist or they are already VOIDED.
func (uc *applyUsecase) ReleaseUsage(ctx context.Context, orderID string) error {
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return uc.ReleaseUsageInTx(ctx, tx, orderID)
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "promotion: ReleaseUsage failed", "order_id", orderID, "err", txErr)
		return txErr
	}
	slog.InfoContext(ctx, "promotion: usages released", "order_id", orderID)
	return nil
}

// ReleaseUsageInTx performs the void + used_count decrement inside a
// caller-owned transaction.
func (uc *applyUsecase) ReleaseUsageInTx(ctx context.Context, tx *gorm.DB, orderID string) error {
	voided, err := uc.usageRepo.VoidByOrderID(ctx, tx, orderID)
	if err != nil {
		return err
	}
	for _, u := range voided {
		if err := uc.promoRepo.DecrementUsedCount(ctx, tx, u.PromotionID); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseExpired sweeps orphaned RESERVED reservations past their ttl and
// releases each via the same idempotent ReleaseUsage path (VOID + decrement
// used_count). Bounded batch per sweep; leftovers are picked up next tick.
func (uc *applyUsecase) ReleaseExpired(ctx context.Context, ttl time.Duration) (int, error) {
	const batchSize = 100
	orderIDs, err := uc.usageRepo.ExpiredReservedOrderIDs(ctx, time.Now().Add(-ttl), batchSize)
	if err != nil {
		return 0, err
	}
	released := 0
	for _, orderID := range orderIDs {
		if err := uc.ReleaseUsage(ctx, orderID); err != nil {
			// Keep sweeping the rest; this order retries on the next tick.
			slog.ErrorContext(ctx, "promotion: expired-reservation release failed", "order_id", orderID, "err", err)
			continue
		}
		released++
	}
	return released, nil
}

// errRollback is a sentinel that signals an intentional rollback. The caller
// inspects `result` for the user-facing error reason; this error is not surfaced.
var errRollback = &rollbackSentinel{}

type rollbackSentinel struct{}

func (e *rollbackSentinel) Error() string { return "rollback" }

func failResult(reason string) *ApplyResult {
	return &ApplyResult{Success: false, ErrorReason: reason}
}

// buildResultFromExisting reconstructs the discount breakdown from persisted
// usage rows (used for idempotent re-apply on the same order_id).
func buildResultFromExisting(usages []*entity.PromotionUsage) *ApplyResult {
	var itemDiscount, shipDiscount int64
	applied := make([]AppliedCode, 0, len(usages))
	for _, u := range usages {
		applied = append(applied, AppliedCode{
			Code:   u.Code,
			Type:   u.Type,
			Amount: u.AppliedAmount,
		})
		// Split by stored type so idempotent re-apply returns the same per-type
		// breakdown the original call did.
		switch u.Type {
		case "SHIP_DISCOUNT":
			shipDiscount += u.AppliedAmount
		default:
			itemDiscount += u.AppliedAmount
		}
	}
	return &ApplyResult{
		Success:      true,
		ItemDiscount: itemDiscount,
		ShipDiscount: shipDiscount,
		Applied:      applied,
	}
}
