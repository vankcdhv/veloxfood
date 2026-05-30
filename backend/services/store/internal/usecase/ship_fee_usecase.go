package usecase

import (
	"context"
	"errors"
	"fmt"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

// RoomResolver resolves a room ID to its parent building ID via the location service.
// Implemented by the location gRPC client in the infrastructure layer.
type RoomResolver interface {
	GetRoom(ctx context.Context, roomID string) (buildingID string, found bool, err error)
}

// ShipFeeUsecase manages ship fee rules and fee resolution.
type ShipFeeUsecase interface {
	CreateShipFeeRule(ctx context.Context, storeID, scope, refID string, unitFee int64) (*entity.ShipFeeRule, error)
	ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error)
	UpdateShipFeeRule(ctx context.Context, id string, unitFee int64) (*entity.ShipFeeRule, error)
	DeleteShipFeeRule(ctx context.Context, id string) error

	// ResolveFee returns the applicable unit fee for a delivery to roomID from storeID.
	// Returns ErrLocationNotServed when no matching rule exists.
	ResolveFee(ctx context.Context, storeID, roomID string) (int64, error)
}

type shipFeeUsecase struct {
	shippingRepo repository.ShippingRepository
	roomResolver RoomResolver
}

func NewShipFeeUsecase(shippingRepo repository.ShippingRepository, roomResolver RoomResolver) ShipFeeUsecase {
	return &shipFeeUsecase{shippingRepo: shippingRepo, roomResolver: roomResolver}
}

func (uc *shipFeeUsecase) CreateShipFeeRule(ctx context.Context, storeID, scope, refID string, unitFee int64) (*entity.ShipFeeRule, error) {
	r := &entity.ShipFeeRule{StoreID: storeID, Scope: scope, RefID: refID, UnitFee: unitFee}
	if err := uc.shippingRepo.CreateShipFeeRule(ctx, r); err != nil {
		return nil, fmt.Errorf("create ship fee rule: %w", err)
	}
	return r, nil
}

func (uc *shipFeeUsecase) ListShipFeeRules(ctx context.Context, storeID string) ([]*entity.ShipFeeRule, error) {
	return uc.shippingRepo.ListShipFeeRules(ctx, storeID)
}

func (uc *shipFeeUsecase) UpdateShipFeeRule(ctx context.Context, id string, unitFee int64) (*entity.ShipFeeRule, error) {
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

// ResolveFee calls the location service to get the building, then looks up the
// most specific fee rule (room > building). Returns ErrLocationNotServed when
// no rule matches.
func (uc *shipFeeUsecase) ResolveFee(ctx context.Context, storeID, roomID string) (int64, error) {
	buildingID, found, err := uc.roomResolver.GetRoom(ctx, roomID)
	if err != nil {
		return 0, fmt.Errorf("resolve room: %w", err)
	}
	if !found {
		return 0, ErrLocationNotServed
	}

	rule, err := uc.shippingRepo.ResolveShipFee(ctx, storeID, buildingID, roomID)
	if err != nil {
		return 0, fmt.Errorf("resolve ship fee: %w", err)
	}
	if rule == nil {
		return 0, ErrLocationNotServed
	}
	return rule.UnitFee, nil
}
