package persistence

import (
	"context"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type vendorMembershipGormRepository struct {
	db *gorm.DB
}

// NewVendorMembershipGormRepository returns a VendorMembershipRepository backed by GORM.
func NewVendorMembershipGormRepository(db *gorm.DB) repository.VendorMembershipRepository {
	return &vendorMembershipGormRepository{db: db}
}

func (r *vendorMembershipGormRepository) Create(ctx context.Context, m *entity.VendorMembership) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *vendorMembershipGormRepository) GetByUserAndVendor(ctx context.Context, userID, vendorID string) (*entity.VendorMembership, error) {
	var m entity.VendorMembership
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND vendor_id = ?", userID, vendorID).
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *vendorMembershipGormRepository) ListByVendor(ctx context.Context, vendorID string, status *entity.VendorMembershipStatus) ([]*entity.VendorMembership, error) {
	q := r.db.WithContext(ctx).Where("vendor_id = ?", vendorID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var members []*entity.VendorMembership
	if err := q.Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *vendorMembershipGormRepository) ListByUser(ctx context.Context, userID string) ([]*entity.VendorMembership, error) {
	var members []*entity.VendorMembership
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *vendorMembershipGormRepository) UpdateStatus(ctx context.Context, id string, status entity.VendorMembershipStatus) error {
	updates := map[string]any{"status": status}
	if status == entity.VendorMembershipActive {
		now := time.Now()
		updates["joined_at"] = now
	}
	return r.db.WithContext(ctx).
		Model(&entity.VendorMembership{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpsertActiveMembership inserts or updates membership using ON CONFLICT (user_id, vendor_id).
// Sets status=active, role_in_vendor, joined_at=now on conflict.
func (r *vendorMembershipGormRepository) UpsertActiveMembership(ctx context.Context, m *entity.VendorMembership) error {
	now := time.Now()
	m.JoinedAt = &now
	m.Status = entity.VendorMembershipActive

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "vendor_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"status":         entity.VendorMembershipActive,
				"role_in_vendor": m.RoleInVendor,
				"joined_at":      now,
			}),
		}).
		Create(m).Error
}

func (r *vendorMembershipGormRepository) CountOwners(ctx context.Context, vendorID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.VendorMembership{}).
		Where("vendor_id = ? AND role_in_vendor = ? AND status = ?",
			vendorID, entity.RoleInVendorOwner, entity.VendorMembershipActive).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
