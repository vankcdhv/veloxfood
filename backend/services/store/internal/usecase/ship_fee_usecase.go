package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

// isDuplicateError detects a Postgres unique-violation (SQLSTATE 23505) so a
// duplicate ship-fee rule surfaces as 409 Conflict instead of a generic 500.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "duplicate")
}

// LocationResolver resolves location IDs across the 3-level hierarchy
// (Building > Floor > Room) via the location service.
// Implemented by the location gRPC client in the infrastructure layer.
type LocationResolver interface {
	// GetRoom resolves roomID to its parent floor and building.
	// Returns (floorID, buildingID, found, err).
	GetRoom(ctx context.Context, roomID string) (floorID, buildingID string, found bool, err error)

	// GetFloor resolves floorID to its parent building.
	// Returns (buildingID, found, err).
	GetFloor(ctx context.Context, floorID string) (buildingID string, found bool, err error)

	// GetBuilding checks whether a building exists.
	// Returns (found, err).
	GetBuilding(ctx context.Context, buildingID string) (found bool, err error)
}

// ShipFeeUsecase manages ship fee rules and fee resolution.
type ShipFeeUsecase interface {
	CreateShipFeeRule(ctx context.Context, storeID, scope, refID string, unitFee int64) (*entity.ShipFeeRule, error)
	ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error)
	UpdateShipFeeRule(ctx context.Context, id string, unitFee int64) (*entity.ShipFeeRule, error)
	DeleteShipFeeRule(ctx context.Context, id string) error

	// ResolveFee returns the applicable unit ship fee for a delivery to the given
	// location (identified by level + locationID) from storeID.
	// level: "ROOM" | "FLOOR" | "BUILDING" (empty defaults to "ROOM").
	// Returns ErrLocationNotServed when no matching rule exists.
	ResolveFee(ctx context.Context, storeID, level, locationID string) (int64, error)
}

type shipFeeUsecase struct {
	shippingRepo     repository.ShippingRepository
	locationResolver LocationResolver
}

func NewShipFeeUsecase(shippingRepo repository.ShippingRepository, locationResolver LocationResolver) ShipFeeUsecase {
	return &shipFeeUsecase{shippingRepo: shippingRepo, locationResolver: locationResolver}
}

// CreateShipFeeRule creates a new ship fee rule.
// Validates "building price first": when scope is "floor" or "room", a building-scope
// rule must already exist for the ancestor building; returns ErrBuildingRuleRequired
// when the prerequisite is missing. unit_fee >= 0 is valid (0 = free delivery).
func (uc *shipFeeUsecase) CreateShipFeeRule(ctx context.Context, storeID, scope, refID string, unitFee int64) (*entity.ShipFeeRule, error) {
	if unitFee < 0 {
		return nil, ErrInvalidUnitFee
	}

	switch scope {
	case "building":
		// No ancestor check needed for building-scope rules.
	case "floor":
		// Require an existing building rule for the ancestor building.
		buildingID, found, err := uc.locationResolver.GetFloor(ctx, refID)
		if err != nil {
			return nil, fmt.Errorf("resolve floor ancestry: %w", err)
		}
		if !found {
			return nil, ErrLocationNotServed
		}
		if err := uc.requireBuildingRule(ctx, storeID, buildingID); err != nil {
			return nil, err
		}
	case "room":
		// Require an existing building rule for the ancestor building.
		_, buildingID, found, err := uc.locationResolver.GetRoom(ctx, refID)
		if err != nil {
			return nil, fmt.Errorf("resolve room ancestry: %w", err)
		}
		if !found {
			return nil, ErrLocationNotServed
		}
		if err := uc.requireBuildingRule(ctx, storeID, buildingID); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidScope
	}

	r := &entity.ShipFeeRule{StoreID: storeID, Scope: scope, RefID: refID, UnitFee: unitFee}
	if err := uc.shippingRepo.CreateShipFeeRule(ctx, r); err != nil {
		if isDuplicateError(err) {
			return nil, ErrShipFeeRuleExists
		}
		return nil, fmt.Errorf("create ship fee rule: %w", err)
	}
	return r, nil
}

// requireBuildingRule returns ErrBuildingRuleRequired if no active building-scope
// rule exists for buildingID in the given store.
func (uc *shipFeeUsecase) requireBuildingRule(ctx context.Context, storeID, buildingID string) error {
	rules, err := uc.shippingRepo.ListShipFeeRules(ctx, storeID)
	if err != nil {
		return fmt.Errorf("list ship fee rules: %w", err)
	}
	for _, r := range rules {
		if r.Scope == "building" && r.RefID == buildingID {
			return nil
		}
	}
	return ErrBuildingRuleRequired
}

func (uc *shipFeeUsecase) ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error) {
	return uc.shippingRepo.ListShipFeeRules(ctx, storeID)
}

func (uc *shipFeeUsecase) UpdateShipFeeRule(ctx context.Context, id string, unitFee int64) (*entity.ShipFeeRule, error) {
	if unitFee < 0 {
		return nil, ErrInvalidUnitFee
	}
	r, err := uc.shippingRepo.GetShipFeeRule(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrShipFeeRuleNotFound
	}
	if err != nil {
		return nil, err
	}
	r.UnitFee = unitFee
	if err := uc.shippingRepo.UpdateShipFeeRule(ctx, r); err != nil {
		return nil, fmt.Errorf("update ship fee rule: %w", err)
	}
	return r, nil
}

func (uc *shipFeeUsecase) DeleteShipFeeRule(ctx context.Context, id string) error {
	return uc.shippingRepo.DeleteShipFeeRule(ctx, id)
}

// ResolveFee resolves the applicable ship fee using the 3-level cascade:
//
//	ROOM     → try rule(room, roomID) > rule(floor, floorID) > rule(building, buildingID)
//	FLOOR    → try rule(floor, floorID) > rule(building, buildingID)
//	BUILDING → try rule(building, buildingID)
//
// Empty level defaults to ROOM for backward compatibility.
// Returns ErrLocationNotServed when no rule matches.
func (uc *shipFeeUsecase) ResolveFee(ctx context.Context, storeID, level, locationID string) (int64, error) {
	if level == "" {
		level = "ROOM"
	}

	// candidates is an ordered list of (scope, id) pairs from most-specific to
	// least-specific. The repo picks the first match in DB.
	type candidate struct{ scope, id string }
	var candidates []candidate

	switch level {
	case "ROOM":
		floorID, buildingID, found, err := uc.locationResolver.GetRoom(ctx, locationID)
		if err != nil {
			return 0, fmt.Errorf("resolve room: %w", err)
		}
		if !found {
			return 0, ErrLocationNotServed
		}
		candidates = []candidate{
			{"room", locationID},
			{"floor", floorID},
			{"building", buildingID},
		}

	case "FLOOR":
		buildingID, found, err := uc.locationResolver.GetFloor(ctx, locationID)
		if err != nil {
			return 0, fmt.Errorf("resolve floor: %w", err)
		}
		if !found {
			return 0, ErrLocationNotServed
		}
		candidates = []candidate{
			{"floor", locationID},
			{"building", buildingID},
		}

	case "BUILDING":
		found, err := uc.locationResolver.GetBuilding(ctx, locationID)
		if err != nil {
			return 0, fmt.Errorf("resolve building: %w", err)
		}
		if !found {
			return 0, ErrLocationNotServed
		}
		candidates = []candidate{
			{"building", locationID},
		}

	default:
		return 0, ErrInvalidLocationLevel
	}

	// Build ordered scope/id slices for the repo; skip empty IDs (e.g. room
	// in a building that has no floor subdivision).
	var scopes, ids []string
	for _, c := range candidates {
		if c.id != "" {
			scopes = append(scopes, c.scope)
			ids = append(ids, c.id)
		}
	}

	rule, err := uc.shippingRepo.ResolveShipFee(ctx, storeID, scopes, ids)
	if err != nil {
		return 0, fmt.Errorf("resolve ship fee: %w", err)
	}
	if rule == nil {
		return 0, ErrLocationNotServed
	}
	return rule.UnitFee, nil
}
