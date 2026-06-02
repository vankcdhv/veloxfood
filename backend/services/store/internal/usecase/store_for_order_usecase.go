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

// CutoffInfo is one of the store's delivery sessions (ca).
type CutoffInfo struct {
	ID          string
	CutoffTime  string // HH:MM
	LeadMinutes int
}

// StoreForOrderResult is the assembled response for an order-time store query.
type StoreForOrderResult struct {
	Found         bool
	SaleStatus    string
	UnitShipFee   int64
	Served        bool
	Items         []OrderItem
	OrderDeadline string // ISO-8601; latest (cutoff - lead_minutes) for today
	Cutoffs       []CutoffInfo
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
}

func NewStoreForOrderUsecase(
	storeRepo repository.StoreRepository,
	catalogRepo repository.CatalogRepository,
	shippingRepo repository.ShippingRepository,
	locationResolver LocationResolver,
) StoreForOrderUsecase {
	return &storeForOrderUsecase{
		storeRepo:        storeRepo,
		catalogRepo:      catalogRepo,
		shippingRepo:     shippingRepo,
		locationResolver: locationResolver,
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
		Found:      true,
		SaleStatus: store.SaleStatus,
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

	// Order deadline = latest (cutoff_time - lead_minutes) among today's cutoffs.
	result.OrderDeadline = resolveOrderDeadline(ctx, uc.shippingRepo, storeID)

	// Expose the raw cutoffs so the order service can compute each item's actual
	// per-slot deadline (date + cutoff_time - lead) and snapshot it.
	if cutoffs, err := uc.shippingRepo.ListShipCutoffs(ctx, storeID); err == nil {
		result.Cutoffs = make([]CutoffInfo, len(cutoffs))
		for i, c := range cutoffs {
			result.Cutoffs[i] = CutoffInfo{ID: c.ID, CutoffTime: c.CutoffTime, LeadMinutes: c.LeadMinutes}
		}
	}

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

// resolveOrderDeadline computes the LATEST order deadline across all of today's
// cutoffs — i.e. the last moment a same-day order can still make some batch.
// With multiple sessions (e.g. 11:00 / 17:30 / 22:00) a customer may keep
// ordering until the final session's deadline; using the earliest cutoff here
// would wrongly close ordering right after the first morning batch.
// Returns empty string when no cutoffs are configured.
func resolveOrderDeadline(ctx context.Context, repo repository.ShippingRepository, storeID string) string {
	cutoffs, err := repo.ListShipCutoffs(ctx, storeID)
	if err != nil || len(cutoffs) == 0 {
		return ""
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var latest *time.Time

	for _, c := range cutoffs {
		dl := parseCutoffDeadline(today, c)
		if dl == nil {
			continue
		}
		if latest == nil || dl.After(*latest) {
			latest = dl
		}
	}

	if latest == nil {
		return ""
	}
	return latest.Format(time.RFC3339)
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
