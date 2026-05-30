package usecase

import (
	"context"
	"errors"

	"project/services/location/internal/entity"
	"project/services/location/internal/repository"

	"gorm.io/gorm"
)

// ResolvedRoom is the full delivery path for a room (for gRPC consumers).
type ResolvedRoom struct {
	Room     *entity.Room
	Floor    *entity.Floor
	Building *entity.Building
}

// LocationUsecase covers admin tree management + public browse + room resolve.
type LocationUsecase interface {
	// Buildings
	CreateBuilding(ctx context.Context, name, address string) (*entity.Building, error)
	ListBuildings(ctx context.Context, activeOnly bool) ([]*entity.Building, error)
	UpdateBuilding(ctx context.Context, id, name, address string, isActive bool) (*entity.Building, error)
	DeleteBuilding(ctx context.Context, id string) error
	// Floors
	CreateFloor(ctx context.Context, buildingID, name string, sortOrder int) (*entity.Floor, error)
	ListFloors(ctx context.Context, buildingID string) ([]*entity.Floor, error)
	UpdateFloor(ctx context.Context, id, name string, sortOrder int) (*entity.Floor, error)
	DeleteFloor(ctx context.Context, id string) error
	// Rooms
	CreateRoom(ctx context.Context, floorID, code, name string) (*entity.Room, error)
	ListRooms(ctx context.Context, floorID string, activeOnly bool) ([]*entity.Room, error)
	UpdateRoom(ctx context.Context, id, code, name string, isActive bool) (*entity.Room, error)
	DeleteRoom(ctx context.Context, id string) error
	// Resolve (gRPC)
	ResolveRoom(ctx context.Context, roomID string) (*ResolvedRoom, error)
}

type locationUsecase struct {
	repo repository.LocationRepository
}

func NewLocationUsecase(repo repository.LocationRepository) LocationUsecase {
	return &locationUsecase{repo: repo}
}

// ---- Buildings ----

func (uc *locationUsecase) CreateBuilding(ctx context.Context, name, address string) (*entity.Building, error) {
	if name == "" {
		return nil, ErrBuildingNameRequired
	}
	b := &entity.Building{Name: name, Address: address, IsActive: true}
	if err := uc.repo.CreateBuilding(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (uc *locationUsecase) ListBuildings(ctx context.Context, activeOnly bool) ([]*entity.Building, error) {
	return uc.repo.ListBuildings(ctx, activeOnly)
}

func (uc *locationUsecase) UpdateBuilding(ctx context.Context, id, name, address string, isActive bool) (*entity.Building, error) {
	b, err := uc.repo.GetBuilding(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, ErrBuildingNotFound)
	}
	if name != "" {
		b.Name = name
	}
	b.Address = address
	b.IsActive = isActive
	if err := uc.repo.UpdateBuilding(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (uc *locationUsecase) DeleteBuilding(ctx context.Context, id string) error {
	return uc.repo.DeleteBuilding(ctx, id)
}

// ---- Floors ----

func (uc *locationUsecase) CreateFloor(ctx context.Context, buildingID, name string, sortOrder int) (*entity.Floor, error) {
	if name == "" {
		return nil, ErrFloorNameRequired
	}
	if _, err := uc.repo.GetBuilding(ctx, buildingID); err != nil {
		return nil, mapNotFound(err, ErrBuildingNotFound)
	}
	f := &entity.Floor{BuildingID: buildingID, Name: name, SortOrder: sortOrder}
	if err := uc.repo.CreateFloor(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (uc *locationUsecase) ListFloors(ctx context.Context, buildingID string) ([]*entity.Floor, error) {
	return uc.repo.ListFloors(ctx, buildingID, false)
}

func (uc *locationUsecase) UpdateFloor(ctx context.Context, id, name string, sortOrder int) (*entity.Floor, error) {
	f, err := uc.repo.GetFloor(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, ErrFloorNotFound)
	}
	if name != "" {
		f.Name = name
	}
	f.SortOrder = sortOrder
	if err := uc.repo.UpdateFloor(ctx, f); err != nil {
		return nil, err
	}
	return f, nil
}

func (uc *locationUsecase) DeleteFloor(ctx context.Context, id string) error {
	return uc.repo.DeleteFloor(ctx, id)
}

// ---- Rooms ----

func (uc *locationUsecase) CreateRoom(ctx context.Context, floorID, code, name string) (*entity.Room, error) {
	if code == "" {
		return nil, ErrRoomCodeRequired
	}
	if _, err := uc.repo.GetFloor(ctx, floorID); err != nil {
		return nil, mapNotFound(err, ErrFloorNotFound)
	}
	room := &entity.Room{FloorID: floorID, Code: code, Name: name, IsActive: true}
	if err := uc.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (uc *locationUsecase) ListRooms(ctx context.Context, floorID string, activeOnly bool) ([]*entity.Room, error) {
	return uc.repo.ListRooms(ctx, floorID, activeOnly)
}

func (uc *locationUsecase) UpdateRoom(ctx context.Context, id, code, name string, isActive bool) (*entity.Room, error) {
	room, err := uc.repo.GetRoom(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, ErrRoomNotFound)
	}
	if code != "" {
		room.Code = code
	}
	room.Name = name
	room.IsActive = isActive
	if err := uc.repo.UpdateRoom(ctx, room); err != nil {
		return nil, err
	}
	return room, nil
}

func (uc *locationUsecase) DeleteRoom(ctx context.Context, id string) error {
	return uc.repo.DeleteRoom(ctx, id)
}

// ---- Resolve ----

func (uc *locationUsecase) ResolveRoom(ctx context.Context, roomID string) (*ResolvedRoom, error) {
	if roomID == "" {
		return nil, ErrRoomIDRequired
	}
	room, err := uc.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, mapNotFound(err, ErrRoomNotFound)
	}
	floor, err := uc.repo.GetFloor(ctx, room.FloorID)
	if err != nil {
		return nil, mapNotFound(err, ErrFloorNotFound)
	}
	building, err := uc.repo.GetBuilding(ctx, floor.BuildingID)
	if err != nil {
		return nil, mapNotFound(err, ErrBuildingNotFound)
	}
	return &ResolvedRoom{Room: room, Floor: floor, Building: building}, nil
}

func mapNotFound(err, notFound error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound
	}
	return err
}
