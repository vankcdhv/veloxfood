package persistence

import (
	"context"
	"errors"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type facultyProfileGormRepository struct {
	db *gorm.DB
}

// NewFacultyProfileGormRepository constructs a GORM-backed FacultyProfileRepository.
func NewFacultyProfileGormRepository(db *gorm.DB) repository.FacultyProfileRepository {
	return &facultyProfileGormRepository{db: db}
}

func (r *facultyProfileGormRepository) GetByUserID(ctx context.Context, userID string) (*entity.FacultyProfile, error) {
	var p entity.FacultyProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &p, nil
}

// Upsert uses PostgreSQL ON CONFLICT (user_id) DO UPDATE.
// Self-service field: allow_payroll_deduction. Admin fields (staff_code, department, position, synced_at)
// included so Phase 06 admin endpoints can reuse this method.
func (r *facultyProfileGormRepository) Upsert(ctx context.Context, profile *entity.FacultyProfile) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"staff_code", "department", "position",
				"allow_payroll_deduction", "synced_at",
			}),
		}).
		Create(profile).Error
}
