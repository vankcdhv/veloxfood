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
	// CheckExpected enforces that the reservation grants exactly
	// ExpectedDiscount (the caller's pre-saga quote). Any drift — e.g. the
	// voucher ran out between quote and apply — fails the request so a DTM
	// saga rolls back instead of charging a price the customer never saw.
	ExpectedDiscount int64
	CheckExpected    bool
}

// ApplyUsecase handles the saga-participant gRPC operations.
type ApplyUsecase interface {
	// ApplyPromotion reserves one or more promotion codes for an order.
	// Idempotent: re-sending the same order_id returns the existing breakdown.
	ApplyPromotion(ctx context.Context, req ApplyRequest) (*ApplyResult, error)

	// QuotePromotion computes the exact breakdown ApplyPromotion would grant,
	// without reserving quota. Order quotes the price before opening a saga.
	QuotePromotion(ctx context.Context, req ApplyRequest) (*ApplyResult, error)

	// ApplyPromotionInTx is ApplyPromotion inside a caller-owned transaction —
	// DTM branch handlers run it under the sub-transaction barrier. A business
	// rejection returns Success=false with a nil error; the caller must then
	// abort its transaction to undo partial reservations.
	ApplyPromotionInTx(ctx context.Context, tx *gorm.DB, req ApplyRequest) (*ApplyResult, error)

	// ConfirmUsage transitions RESERVED usages for an order to CONFIRMED.
	ConfirmUsage(ctx context.Context, orderID string) error

	// ConfirmUsageInTx is ConfirmUsage inside a caller-owned transaction.
	ConfirmUsageInTx(ctx context.Context, tx *gorm.DB, orderID string) error

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
	var result *ApplyResult
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		r, err := uc.ApplyPromotionInTx(ctx, tx, req)
		if err != nil {
			return err
		}
		result = r
		if !r.Success {
			// Roll back partial reservations; the reason travels in `result`.
			return errRollback
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

// ApplyPromotionInTx reserves the requested codes inside a caller-owned
// transaction. Business rejections come back as Success=false (nil error) —
// the caller decides how to abort so partial usage inserts roll back with it.
func (uc *applyUsecase) ApplyPromotionInTx(ctx context.Context, tx *gorm.DB, req ApplyRequest) (*ApplyResult, error) {
	// Idempotency: if usages already exist for this order, return the existing
	// breakdown (still subject to the expected-discount check below).
	existing, err := uc.usageRepo.ListByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return checkExpectedDiscount(buildResultFromExisting(existing), req), nil
	}

	now := time.Now()
	var appliedCodes []AppliedCode
	var itemDiscount, shipDiscount int64

	// Track types already granted in this request to enforce ≤1 per type.
	grantedTypes := map[string]bool{}

	for _, code := range req.Codes {
		p, err := uc.promoRepo.GetByStoreAndCodeForUpdate(ctx, tx, req.StoreID, code)
		if err != nil {
			return failResult("promotion not found: " + code), nil
		}

		if reason := validatePromotion(p, req.StoreID, req.Subtotal, now); reason != "" {
			return failResult(reason), nil
		}

		// Enforce at most one ORDER_DISCOUNT and one SHIP_DISCOUNT per order.
		if grantedTypes[p.Type] {
			return failResult("cannot apply two " + p.Type + " promotions to one order"), nil
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
			return failResult("failed to reserve promotion"), nil
		}

		if err := uc.promoRepo.IncrementUsedCount(ctx, tx, p.ID); err != nil {
			return failResult("failed to reserve promotion"), nil
		}

		appliedCodes = append(appliedCodes, AppliedCode{Code: code, Type: p.Type, Amount: amount})
		switch p.Type {
		case "ORDER_DISCOUNT":
			itemDiscount += amount
		case "SHIP_DISCOUNT":
			shipDiscount += amount
		}
	}

	return checkExpectedDiscount(&ApplyResult{
		Success:      true,
		ItemDiscount: itemDiscount,
		ShipDiscount: shipDiscount,
		Applied:      appliedCodes,
	}, req), nil
}

// checkExpectedDiscount fails a successful reservation whose total discount
// drifted from the caller's quote (voucher state changed between quote and
// apply). No-op unless the request opted in.
func checkExpectedDiscount(r *ApplyResult, req ApplyRequest) *ApplyResult {
	if !req.CheckExpected || !r.Success {
		return r
	}
	if r.ItemDiscount+r.ShipDiscount != req.ExpectedDiscount {
		return failResult("discount changed since quote — please retry the order")
	}
	return r
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
	err := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return uc.ConfirmUsageInTx(ctx, tx, orderID)
	})
	if err != nil {
		slog.ErrorContext(ctx, "promotion: ConfirmUsage failed", "order_id", orderID, "err", err)
		return err
	}
	slog.InfoContext(ctx, "promotion: usages confirmed", "order_id", orderID)
	return nil
}

// ConfirmUsageInTx performs the RESERVED→CONFIRMED transition inside a
// caller-owned transaction (DTM branch handlers run it under the barrier).
func (uc *applyUsecase) ConfirmUsageInTx(ctx context.Context, tx *gorm.DB, orderID string) error {
	return uc.usageRepo.ConfirmByOrderID(ctx, tx, orderID)
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
