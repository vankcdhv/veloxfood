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
	Add(ctx context.Context, customerID, locationID, level, label string, makeDefault bool) (*entity.CustomerLocation, error)
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
		uc.resolvePath(ctx, v)
		views = append(views, v)
	}
	return views, nil
}

// resolvePath fills the readable building/floor/room names according to the
// saved location's level. A missing node (e.g. soft-deleted) leaves the
// corresponding name blank rather than failing.
func (uc *customerLocationUsecase) resolvePath(ctx context.Context, v *CustomerLocationView) {
	switch v.LocationLevel {
	case entity.LocationLevelBuilding:
		if b, err := uc.locRepo.GetBuilding(ctx, v.LocationID); err == nil {
			v.BuildingName = b.Name
		}
	case entity.LocationLevelFloor:
		if floor, err := uc.locRepo.GetFloor(ctx, v.LocationID); err == nil {
			v.FloorName = floor.Name
			if b, err := uc.locRepo.GetBuilding(ctx, floor.BuildingID); err == nil {
				v.BuildingName = b.Name
			}
		}
	default: // ROOM
		if room, err := uc.locRepo.GetRoom(ctx, v.LocationID); err == nil {
			v.RoomCode, v.RoomName = room.Code, room.Name
			if floor, err := uc.locRepo.GetFloor(ctx, room.FloorID); err == nil {
				v.FloorName = floor.Name
				if b, err := uc.locRepo.GetBuilding(ctx, floor.BuildingID); err == nil {
					v.BuildingName = b.Name
				}
			}
		}
	}
}

func (uc *customerLocationUsecase) Add(ctx context.Context, customerID, locationID, level, label string, makeDefault bool) (*entity.CustomerLocation, error) {
	if locationID == "" {
		return nil, ErrLocationIDRequired
	}
	// Validate the referenced node exists at the claimed level.
	switch level {
	case entity.LocationLevelBuilding:
		if _, err := uc.locRepo.GetBuilding(ctx, locationID); err != nil {
			return nil, mapNotFound(err, ErrBuildingNotFound)
		}
	case entity.LocationLevelFloor:
		if _, err := uc.locRepo.GetFloor(ctx, locationID); err != nil {
			return nil, mapNotFound(err, ErrFloorNotFound)
		}
	case entity.LocationLevelRoom:
		if _, err := uc.locRepo.GetRoom(ctx, locationID); err != nil {
			return nil, mapNotFound(err, ErrRoomNotFound)
		}
	default:
		return nil, ErrInvalidLocationLevel
	}
	loc := &entity.CustomerLocation{CustomerID: customerID, LocationLevel: level, LocationID: locationID, Label: label}
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
