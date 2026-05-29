package persistence

import (
	"context"
	"errors"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type studentProfileGormRepository struct {
	db *gorm.DB
}

// NewStudentProfileGormRepository constructs a GORM-backed StudentProfileRepository.
func NewStudentProfileGormRepository(db *gorm.DB) repository.StudentProfileRepository {
	return &studentProfileGormRepository{db: db}
}

func (r *studentProfileGormRepository) GetByUserID(ctx context.Context, userID string) (*entity.StudentProfile, error) {
	var p entity.StudentProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &p, nil
}

// Upsert uses PostgreSQL ON CONFLICT (user_id) DO UPDATE to insert or replace
// the mutable self-service fields: allergies and dormitory_room.
// Admin-owned fields (student_code, faculty, class, cohort_year, synced_at) are
// also updated here so admin endpoints (Phase 06) can reuse this method.
func (r *studentProfileGormRepository) Upsert(ctx context.Context, profile *entity.StudentProfile) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"student_code", "faculty", "class", "cohort_year",
				"allergies", "dormitory_room", "synced_at",
			}),
		}).
		Create(profile).Error
}
