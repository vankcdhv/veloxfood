package usecase

import (
	"context"

	"project/services/location/internal/entity"
	"project/services/location/internal/repository"
)

// CustomerLocationView is a saved location enriched with its human-readable
// delivery path (building/floor/room) so clients never render the raw room UUID.
type CustomerLocationView struct {
	*entity.CustomerLocation
	BuildingName string
	FloorName    string
	RoomCode     string
	RoomName     string
}

// CustomerLocationUsecase manages a customer's saved delivery locations.
type CustomerLocationUsecase interface {
	List(ctx context.Context, customerID string) ([]*CustomerLocationView, error)
	Add(ctx context.Context, customerID, roomID, label string, makeDefault bool) (*entity.CustomerLocation, error)
	Remove(ctx context.Context, customerID, id string) error
	SetDefault(ctx context.Context, customerID, id string) error
}

type customerLocationUsecase struct {
	repo    repository.CustomerLocationRepository
	locRepo repository.LocationRepository
}

func NewCustomerLocationUsecase(repo repository.CustomerLocationRepository, locRepo repository.LocationRepository) CustomerLocationUsecase {
	return &customerLocationUsecase{repo: repo, locRepo: locRepo}
}

func (uc *customerLocationUsecase) List(ctx context.Context, customerID string) ([]*CustomerLocationView, error) {
	locs, err := uc.repo.ListByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	views := make([]*CustomerLocationView, 0, len(locs))
	for _, loc := range locs {
		v := &CustomerLocationView{CustomerLocation: loc}
		// Resolve the readable path; tolerate a missing node (e.g. soft-deleted
		// building) by leaving the corresponding name blank.
		if room, err := uc.locRepo.GetRoom(ctx, loc.RoomID); err == nil {
			v.RoomCode, v.RoomName = room.Code, room.Name
			if floor, err := uc.locRepo.GetFloor(ctx, room.FloorID); err == nil {
				v.FloorName = floor.Name
				if b, err := uc.locRepo.GetBuilding(ctx, floor.BuildingID); err == nil {
					v.BuildingName = b.Name
				}
			}
		}
		views = append(views, v)
	}
	return views, nil
}

func (uc *customerLocationUsecase) Add(ctx context.Context, customerID, roomID, label string, makeDefault bool) (*entity.CustomerLocation, error) {
	if roomID == "" {
		return nil, ErrRoomIDRequired
	}
	// Validate the room exists (and is a real delivery target).
	if _, err := uc.locRepo.GetRoom(ctx, roomID); err != nil {
		return nil, mapNotFound(err, ErrRoomNotFound)
	}
	loc := &entity.CustomerLocation{CustomerID: customerID, RoomID: roomID, Label: label}
	if err := uc.repo.Create(ctx, loc); err != nil {
		return nil, err
	}
	if makeDefault {
		if err := uc.repo.SetDefault(ctx, loc.ID, customerID); err != nil {
			return nil, err
		}
		loc.IsDefault = true
	}
	return loc, nil
}

func (uc *customerLocationUsecase) Remove(ctx context.Context, customerID, id string) error {
	return uc.repo.Delete(ctx, id, customerID)
}

func (uc *customerLocationUsecase) SetDefault(ctx context.Context, customerID, id string) error {
	if _, err := uc.repo.GetByID(ctx, id); err != nil {
		return mapNotFound(err, ErrCustomerLocationNotFound)
	}
	return uc.repo.SetDefault(ctx, id, customerID)
}
