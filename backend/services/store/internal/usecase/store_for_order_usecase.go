package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	Found          bool
	SaleStatus     string
	UnitShipFee    int64
	Served         bool
	Items          []OrderItem
	PrepMinutes    int
	OpenNow        bool
	OpenTimeToday  string // HH:MM; empty when no hours configured for today
	CloseTimeToday string // HH:MM; empty when no hours configured for today
}

// StoreForOrderUsecase is called by the gRPC handler to validate a store
// before order placement.
type StoreForOrderUsecase interface {
	// GetStoreForOrder assembles store availability, ship fee, and menu snapshot.
	// locationLevel: "ROOM" | "FLOOR" | "BUILDING" (empty → ROOM).
	// locationID: the ID at that level; empty for PICKUP orders.
	GetStoreForOrder(ctx context.Context, storeID, locationLevel, locationID string) (*StoreForOrderResult, error)
}

type storeForOrderUsecase struct {
	storeRepo        repository.StoreRepository
	catalogRepo      repository.CatalogRepository
	shippingRepo     repository.ShippingRepository
	locationResolver LocationResolver
	hoursUC          HoursUsecase
}

func NewStoreForOrderUsecase(
	storeRepo repository.StoreRepository,
	catalogRepo repository.CatalogRepository,
	shippingRepo repository.ShippingRepository,
	locationResolver LocationResolver,
	hoursUC HoursUsecase,
) StoreForOrderUsecase {
	return &storeForOrderUsecase{
		storeRepo:        storeRepo,
		catalogRepo:      catalogRepo,
		shippingRepo:     shippingRepo,
		locationResolver: locationResolver,
		hoursUC:          hoursUC,
	}
}

func (uc *storeForOrderUsecase) GetStoreForOrder(ctx context.Context, storeID, locationLevel, locationID string) (*StoreForOrderResult, error) {
	store, err := uc.storeRepo.GetByID(ctx, storeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &StoreForOrderResult{Found: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get store: %w", err)
	}

	result := &StoreForOrderResult{
		Found:       true,
		SaleStatus:  store.SaleStatus,
		PrepMinutes: store.PrepMinutes,
	}

	// Resolve ship fee for delivery. PICKUP orders carry no locationID, so skip
	// resolution entirely (empty locationID is not an error for PICKUP).
	if locationID != "" {
		level := locationLevel
		if level == "" {
			level = "ROOM"
		}
		fee, feeErr := resolveFeeForOrder(ctx, uc.locationResolver, uc.shippingRepo, storeID, level, locationID)
		if feeErr == nil {
			result.Served = true
			result.UnitShipFee = fee
		}
		// ErrLocationNotServed or location-not-found are silently treated as
		// Served=false — the order service may still proceed for PICKUP.
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

	// Populate open-now status for the FE checkout gate.
	openResult, err := uc.hoursUC.IsOpenNow(ctx, storeID, store.SaleStatus, time.Now())
	if err == nil {
		result.OpenNow = openResult.OpenNow
		result.OpenTimeToday = openResult.OpenTimeToday
		result.CloseTimeToday = openResult.CloseTimeToday
	}
	// Tolerate hours lookup failure — result fields stay zero-valued (closed).

	return result, nil
}

// resolveFeeForOrder applies the 3-level cascade to find the applicable ship fee.
// It mirrors ResolveFee in ShipFeeUsecase but operates directly on the repo so
// StoreForOrderUsecase does not depend on ShipFeeUsecase (avoids circular deps).
func resolveFeeForOrder(ctx context.Context, resolver LocationResolver, repo repository.ShippingRepository, storeID, level, locationID string) (int64, error) {
	type candidate struct{ scope, id string }
	var candidates []candidate

	switch level {
	case "ROOM":
		floorID, buildingID, found, err := resolver.GetRoom(ctx, locationID)
		if err != nil || !found {
			return 0, ErrLocationNotServed
		}
		candidates = []candidate{{"room", locationID}, {"floor", floorID}, {"building", buildingID}}
	case "FLOOR":
		buildingID, found, err := resolver.GetFloor(ctx, locationID)
		if err != nil || !found {
			return 0, ErrLocationNotServed
		}
		candidates = []candidate{{"floor", locationID}, {"building", buildingID}}
	case "BUILDING":
		found, err := resolver.GetBuilding(ctx, locationID)
		if err != nil || !found {
			return 0, ErrLocationNotServed
		}
		candidates = []candidate{{"building", locationID}}
	default:
		return 0, ErrLocationNotServed
	}

	var scopes, ids []string
	for _, c := range candidates {
		if c.id != "" {
			scopes = append(scopes, c.scope)
			ids = append(ids, c.id)
		}
	}
	rule, err := repo.ResolveShipFee(ctx, storeID, scopes, ids)
	if err != nil {
		return 0, err
	}
	if rule == nil {
		return 0, ErrLocationNotServed
	}
	return rule.UnitFee, nil
}
