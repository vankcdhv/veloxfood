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

	// ConfirmUsage transitions RESERVED usages for an order to CONFIRMED.
	ConfirmUsage(ctx context.Context, orderID string) error

	// ReleaseUsage voids RESERVED usages for an order and decrements used_count.
	ReleaseUsage(ctx context.Context, orderID string) error
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
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "promotion: ReleaseUsage failed", "order_id", orderID, "err", txErr)
		return txErr
	}
	slog.InfoContext(ctx, "promotion: usages released", "order_id", orderID)
	return nil
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
