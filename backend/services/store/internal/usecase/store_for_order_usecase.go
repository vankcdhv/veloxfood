package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

// OrderItem is a catalog snapshot passed to the order service for validation.
type OrderItem struct {
	ItemID string
	Name   string
	Price  int64
}

// StoreForOrderResult is the assembled response for an order-time store query.
type StoreForOrderResult struct {
	Found         bool
	SaleStatus    string
	UnitShipFee   int64
	Served        bool
	Items         []OrderItem
	OrderDeadline string // ISO-8601; earliest (cutoff - lead_minutes) for today
}

// StoreForOrderUsecase is called by the gRPC handler to validate a store
// before order placement.
type StoreForOrderUsecase interface {
	GetStoreForOrder(ctx context.Context, storeID, roomID string) (*StoreForOrderResult, error)
}

type storeForOrderUsecase struct {
	storeRepo    repository.StoreRepository
	catalogRepo  repository.CatalogRepository
	shippingRepo repository.ShippingRepository
	roomResolver RoomResolver
}

func NewStoreForOrderUsecase(
	storeRepo repository.StoreRepository,
	catalogRepo repository.CatalogRepository,
	shippingRepo repository.ShippingRepository,
	roomResolver RoomResolver,
) StoreForOrderUsecase {
	return &storeForOrderUsecase{
		storeRepo:    storeRepo,
		catalogRepo:  catalogRepo,
		shippingRepo: shippingRepo,
		roomResolver: roomResolver,
	}
}

func (uc *storeForOrderUsecase) GetStoreForOrder(ctx context.Context, storeID, roomID string) (*StoreForOrderResult, error) {
	store, err := uc.storeRepo.GetByID(ctx, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &StoreForOrderResult{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get store: %w", err)
	}

	result := &StoreForOrderResult{
		Found:      true,
		SaleStatus: store.SaleStatus,
	}

	// Resolve ship fee for delivery. PICKUP orders carry no room, so skip
	// resolution entirely (an empty room id is not an error).
	if roomID != "" {
		buildingID, found, err := uc.roomResolver.GetRoom(ctx, roomID)
		if err != nil {
			return nil, fmt.Errorf("resolve room: %w", err)
		}
		if found {
			rule, err := uc.shippingRepo.ResolveShipFee(ctx, storeID, buildingID, roomID)
			if err == nil && rule != nil {
				result.Served = true
				result.UnitShipFee = rule.UnitFee
			}
		}
	}

	// Load active menu items for order validation snapshot.
	items, err := uc.catalogRepo.ListMenuItems(ctx, storeID, "", "on")
	if err != nil {
		return nil, fmt.Errorf("list menu items: %w", err)
	}
	result.Items = make([]OrderItem, len(items))
	for i, m := range items {
		result.Items[i] = OrderItem{ItemID: m.ID, Name: m.Name, Price: m.Price}
	}

	// Order deadline = earliest (cutoff_time - lead_minutes) among today's cutoffs.
	result.OrderDeadline = resolveOrderDeadline(ctx, uc.shippingRepo, storeID)

	return result, nil
}

// resolveOrderDeadline computes the earliest order deadline across all cutoffs for
// the store today. Returns empty string when no cutoffs are configured.
func resolveOrderDeadline(ctx context.Context, repo repository.ShippingRepository, storeID string) string {
	cutoffs, err := repo.ListShipCutoffs(ctx, storeID)
	if err != nil || len(cutoffs) == 0 {
		return ""
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var earliest *time.Time

	for _, c := range cutoffs {
		dl := parseCutoffDeadline(today, c)
		if dl == nil {
			continue
		}
		if earliest == nil || dl.Before(*earliest) {
			earliest = dl
		}
	}

	if earliest == nil {
		return ""
	}
	return earliest.Format(time.RFC3339)
}

// parseCutoffDeadline parses "HH:MM" cutoff time and subtracts lead minutes.
func parseCutoffDeadline(today time.Time, c *entity.ShipCutoff) *time.Time {
	var h, m int
	if _, err := fmt.Sscanf(c.CutoffTime, "%d:%d", &h, &m); err != nil {
		return nil
	}
	cutoff := today.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	deadline := cutoff.Add(-time.Duration(c.LeadMinutes) * time.Minute)
	return &deadline
}
