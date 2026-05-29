package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// StudentProfileRepository is the persistence contract for student profiles.
// UserID is both PK and FK — one profile per user.
type StudentProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*entity.StudentProfile, error)
	// Upsert inserts or updates the profile (ON CONFLICT user_id DO UPDATE).
	Upsert(ctx context.Context, profile *entity.StudentProfile) error
}
