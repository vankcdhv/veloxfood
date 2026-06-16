package usecase

import (
	"context"
	"log/slog"
	"time"

	"project/services/promotion/internal/entity"
	"project/services/promotion/internal/infrastructure/grpcclient"
	"project/services/promotion/internal/repository"

	"gorm.io/gorm"
)

// PromotionUsecase handles owner CRUD and the dry-run validate operation.
type PromotionUsecase interface {
	// CreatePromotion creates a new voucher for a store.
	CreatePromotion(ctx context.Context, req CreatePromotionRequest) (*entity.Promotion, error)

	// ListPromotions returns a page of non-deleted promotions for a store plus total.
	ListPromotions(ctx context.Context, storeID string, limit, offset int) ([]*entity.Promotion, int64, error)

	// GetPromotion returns a single promotion by ID.
	GetPromotion(ctx context.Context, id string) (*entity.Promotion, error)

	// UpdatePromotion mutates mutable fields (code is immutable).
	// storeID is the route-authorised store; the call fails with not-found if the
	// promotion belongs to a different store (prevents cross-tenant edits).
	UpdatePromotion(ctx context.Context, id string, storeID string, req UpdatePromotionRequest) (*entity.Promotion, error)

	// DeletePromotion soft-deletes a promotion.
	// storeID is the route-authorised store; the call fails with not-found if the
	// promotion belongs to a different store (prevents cross-tenant deletes).
	DeletePromotion(ctx context.Context, id string, storeID string) error

	// ValidatePromotion dry-runs the discount calculation without reserving.
	ValidatePromotion(ctx context.Context, req ValidateRequest) (*ValidateResult, error)

	// GetStoreOwnership returns the vendor_id for a store (wraps gRPC client).
	GetStoreOwnership(ctx context.Context, storeID string) (*grpcclient.StoreOwnership, error)
}

// CreatePromotionRequest carries the fields for a new promotion.
type CreatePromotionRequest struct {
	StoreID     string
	Code        string
	Type        string
	ValueKind   string
	Value       int64
	MinOrder    int64
	MaxDiscount *int64
	StartsAt    time.Time
	EndsAt      time.Time
	UsageLimit  *int
}

// UpdatePromotionRequest carries mutable fields (code excluded — immutable).
type UpdatePromotionRequest struct {
	Value       int64
	MinOrder    int64
	MaxDiscount *int64
	StartsAt    time.Time
	EndsAt      time.Time
	UsageLimit  *int
	Status      string
}

// ValidateRequest is the input for a dry-run discount preview.
type ValidateRequest struct {
	Code      string
	StoreID   string
	Subtotal  int64
	ItemCount int32
}

// ValidateResult is the response for a dry-run discount preview.
// PascalCase — no json tags; serialises consistently via response envelope.
type ValidateResult struct {
	Applicable   bool
	ItemDiscount int64
	ShipDiscount int64
	ErrorReason  string
}

var validTypes = map[string]bool{
	"ORDER_DISCOUNT": true,
	"SHIP_DISCOUNT":  true,
}

var validValueKinds = map[string]bool{
	"PERCENT": true,
	"AMOUNT":  true,
}

var validStatuses = map[string]bool{
	"ACTIVE": true, "INACTIVE": true,
}

type promotionUsecase struct {
	db          *gorm.DB
	promoRepo   repository.PromotionRepository
	usageRepo   repository.PromotionUsageRepository
	storeClient grpcclient.StoreClient
}

// NewPromotionUsecase constructs PromotionUsecase.
func NewPromotionUsecase(
	db *gorm.DB,
	promoRepo repository.PromotionRepository,
	usageRepo repository.PromotionUsageRepository,
	storeClient grpcclient.StoreClient,
) PromotionUsecase {
	return &promotionUsecase{
		db:          db,
		promoRepo:   promoRepo,
		usageRepo:   usageRepo,
		storeClient: storeClient,
	}
}

func (uc *promotionUsecase) GetStoreOwnership(ctx context.Context, storeID string) (*grpcclient.StoreOwnership, error) {
	return uc.storeClient.GetStoreOwnership(ctx, storeID)
}

func (uc *promotionUsecase) CreatePromotion(ctx context.Context, req CreatePromotionRequest) (*entity.Promotion, error) {
	if !validTypes[req.Type] {
		return nil, ErrInvalidType
	}
	if !validValueKinds[req.ValueKind] {
		return nil, ErrInvalidValueKind
	}
	if !req.StartsAt.IsZero() && !req.EndsAt.IsZero() && req.StartsAt.After(req.EndsAt) {
		return nil, ErrInvalidTimeframe
	}

	p := &entity.Promotion{
		StoreID:     req.StoreID,
		Code:        req.Code,
		Type:        req.Type,
		ValueKind:   req.ValueKind,
		Value:       req.Value,
		MinOrder:    req.MinOrder,
		MaxDiscount: req.MaxDiscount,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		UsageLimit:  req.UsageLimit,
		Status:      "ACTIVE",
	}

	if err := uc.promoRepo.Create(ctx, p); err != nil {
		slog.ErrorContext(ctx, "promotion: create failed", "store_id", req.StoreID, "code", req.Code, "err", err)
		return nil, err
	}

	slog.InfoContext(ctx, "promotion created", "id", p.ID, "store_id", p.StoreID, "code", p.Code)
	return p, nil
}

func (uc *promotionUsecase) ListPromotions(ctx context.Context, storeID string, limit, offset int) ([]*entity.Promotion, int64, error) {
	return uc.promoRepo.ListByStore(ctx, storeID, limit, offset)
}

func (uc *promotionUsecase) GetPromotion(ctx context.Context, id string) (*entity.Promotion, error) {
	return uc.promoRepo.GetByID(ctx, id)
}

func (uc *promotionUsecase) UpdatePromotion(ctx context.Context, id string, storeID string, req UpdatePromotionRequest) (*entity.Promotion, error) {
	if req.Status != "" && !validStatuses[req.Status] {
		return nil, ErrInvalidStatus
	}
	if !req.StartsAt.IsZero() && !req.EndsAt.IsZero() && req.StartsAt.After(req.EndsAt) {
		return nil, ErrInvalidTimeframe
	}

	p, err := uc.promoRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Scope check: ensure the promotion belongs to the authorised store.
	// Returning not-found (rather than forbidden) avoids leaking whether the
	// promotion exists at all — standard cross-tenant isolation pattern.
	if p.StoreID != storeID {
		return nil, ErrPromotionNotFound
	}

	// Apply only non-zero values so partial PATCH works correctly.
	if req.Value != 0 {
		p.Value = req.Value
	}
	p.MinOrder = req.MinOrder
	p.MaxDiscount = req.MaxDiscount
	if !req.StartsAt.IsZero() {
		p.StartsAt = req.StartsAt
	}
	if !req.EndsAt.IsZero() {
		p.EndsAt = req.EndsAt
	}
	p.UsageLimit = req.UsageLimit
	if req.Status != "" {
		p.Status = req.Status
	}
	p.UpdatedAt = time.Now()

	if err := uc.promoRepo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (uc *promotionUsecase) DeletePromotion(ctx context.Context, id string, storeID string) error {
	// Load first so we can assert store ownership before issuing the DELETE.
	// Returning not-found on mismatch avoids leaking whether the promotion exists
	// at all — standard cross-tenant isolation pattern.
	p, err := uc.promoRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if p.StoreID != storeID {
		return ErrPromotionNotFound
	}
	return uc.promoRepo.Delete(ctx, id)
}

// ValidatePromotion is a dry-run preview — no reservation is created.
func (uc *promotionUsecase) ValidatePromotion(ctx context.Context, req ValidateRequest) (*ValidateResult, error) {
	p, err := uc.promoRepo.GetByStoreAndCodeForUpdate(ctx, uc.db, req.StoreID, req.Code)
	if err != nil {
		return &ValidateResult{Applicable: false, ErrorReason: "promotion not found"}, nil
	}

	now := time.Now()
	if reason := validatePromotion(p, req.StoreID, req.Subtotal, now); reason != "" {
		return &ValidateResult{Applicable: false, ErrorReason: reason}, nil
	}

	amount := computeDiscount(p, req.Subtotal)
	result := &ValidateResult{Applicable: true}
	switch p.Type {
	case "ORDER_DISCOUNT":
		result.ItemDiscount = amount
	case "SHIP_DISCOUNT":
		result.ShipDiscount = amount
	}
	return result, nil
}

// validatePromotion checks all business rules and returns an error reason string.
// Empty string means the promotion is valid.
func validatePromotion(p *entity.Promotion, storeID string, subtotal int64, now time.Time) string {
	if p.StoreID != storeID {
		return "promotion does not belong to this store"
	}
	if p.Status != "ACTIVE" {
		return "promotion is not active"
	}
	if now.Before(p.StartsAt) {
		return "promotion has not started yet"
	}
	if now.After(p.EndsAt) {
		return "promotion has expired"
	}
	if subtotal < p.MinOrder {
		return "order subtotal does not meet the minimum order requirement"
	}
	if p.UsageLimit != nil && p.UsedCount >= *p.UsageLimit {
		return "promotion usage limit has been reached"
	}
	return ""
}

// computeDiscount calculates the discount amount in VND for a promotion.
func computeDiscount(p *entity.Promotion, subtotal int64) int64 {
	switch p.ValueKind {
	case "PERCENT":
		amount := subtotal * p.Value / 100
		if p.MaxDiscount != nil && amount > *p.MaxDiscount {
			amount = *p.MaxDiscount
		}
		return amount
	case "AMOUNT":
		if p.Value > subtotal {
			return subtotal
		}
		return p.Value
	default:
		return 0
	}
}
