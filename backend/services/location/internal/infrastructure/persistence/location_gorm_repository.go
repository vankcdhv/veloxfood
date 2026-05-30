package persistence

import (
	"context"

	"project/services/location/internal/entity"
	"project/services/location/internal/repository"

	"gorm.io/gorm"
)

type locationGormRepository struct {
	db *gorm.DB
}

func NewLocationGormRepository(db *gorm.DB) repository.LocationRepository {
	return &locationGormRepository{db: db}
}

// ---- Buildings ----

func (r *locationGormRepository) CreateBuilding(ctx context.Context, b *entity.Building) error {
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *locationGormRepository) GetBuilding(ctx context.Context, id string) (*entity.Building, error) {
	var b entity.Building
	if err := r.db.WithContext(ctx).First(&b, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *locationGormRepository) ListBuildings(ctx context.Context, activeOnly bool) ([]*entity.Building, error) {
	q := r.db.WithContext(ctx).Order("name ASC")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var rows []*entity.Building
	return rows, q.Find(&rows).Error
}

func (r *locationGormRepository) UpdateBuilding(ctx context.Context, b *entity.Building) error {
	return r.db.WithContext(ctx).Model(b).Updates(map[string]any{
		"name": b.Name, "address": b.Address, "is_active": b.IsActive,
	}).Error
}

func (r *locationGormRepository) DeleteBuilding(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Building{}, "id = ?", id).Error
}

// ---- Floors ----

func (r *locationGormRepository) CreateFloor(ctx context.Context, f *entity.Floor) error {
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *locationGormRepository) GetFloor(ctx context.Context, id string) (*entity.Floor, error) {
	var f entity.Floor
	if err := r.db.WithContext(ctx).First(&f, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *locationGormRepository) ListFloors(ctx context.Context, buildingID string, activeOnly bool) ([]*entity.Floor, error) {
	q := r.db.WithContext(ctx).Where("building_id = ?", buildingID).Order("sort_order ASC, name ASC")
	var rows []*entity.Floor
	_ = activeOnly // floors have no is_active; soft-delete handled by gorm
	return rows, q.Find(&rows).Error
}

func (r *locationGormRepository) UpdateFloor(ctx context.Context, f *entity.Floor) error {
	return r.db.WithContext(ctx).Model(f).Updates(map[string]any{
		"name": f.Name, "sort_order": f.SortOrder,
	}).Error
}

func (r *locationGormRepository) DeleteFloor(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Floor{}, "id = ?", id).Error
}

// ---- Rooms ----

func (r *locationGormRepository) CreateRoom(ctx context.Context, room *entity.Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *locationGormRepository) GetRoom(ctx context.Context, id string) (*entity.Room, error) {
	var room entity.Room
	if err := r.db.WithContext(ctx).First(&room, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *locationGormRepository) ListRooms(ctx context.Context, floorID string, activeOnly bool) ([]*entity.Room, error) {
	q := r.db.WithContext(ctx).Where("floor_id = ?", floorID).Order("code ASC")
	if activeOnly {
		q = q.Where("is_active = ?", true)
	}
	var rows []*entity.Room
	return rows, q.Find(&rows).Error
}

func (r *locationGormRepository) UpdateRoom(ctx context.Context, room *entity.Room) error {
	return r.db.WithContext(ctx).Model(room).Updates(map[string]any{
		"code": room.Code, "name": room.Name, "is_active": room.IsActive,
	}).Error
}

func (r *locationGormRepository) DeleteRoom(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Room{}, "id = ?", id).Error
}
