package repository

import (
	"context"

	"project/services/location/internal/entity"
)

// LocationRepository manages the Building → Floor → Room tree.
type LocationRepository interface {
	// Buildings
	CreateBuilding(ctx context.Context, b *entity.Building) error
	GetBuilding(ctx context.Context, id string) (*entity.Building, error)
	ListBuildings(ctx context.Context, activeOnly bool) ([]*entity.Building, error)
	UpdateBuilding(ctx context.Context, b *entity.Building) error
	DeleteBuilding(ctx context.Context, id string) error // soft-delete

	// Floors
	CreateFloor(ctx context.Context, f *entity.Floor) error
	GetFloor(ctx context.Context, id string) (*entity.Floor, error)
	ListFloors(ctx context.Context, buildingID string, activeOnly bool) ([]*entity.Floor, error)
	UpdateFloor(ctx context.Context, f *entity.Floor) error
	DeleteFloor(ctx context.Context, id string) error

	// Rooms
	CreateRoom(ctx context.Context, r *entity.Room) error
	GetRoom(ctx context.Context, id string) (*entity.Room, error)
	ListRooms(ctx context.Context, floorID string, activeOnly bool) ([]*entity.Room, error)
	UpdateRoom(ctx context.Context, r *entity.Room) error
	DeleteRoom(ctx context.Context, id string) error
}
