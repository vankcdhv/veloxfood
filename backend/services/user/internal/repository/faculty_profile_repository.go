package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// FacultyProfileRepository is the persistence contract for faculty profiles.
// UserID is both PK and FK — one profile per user.
type FacultyProfileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*entity.FacultyProfile, error)
	// Upsert inserts or updates the profile (ON CONFLICT user_id DO UPDATE).
	Upsert(ctx context.Context, profile *entity.FacultyProfile) error
}
