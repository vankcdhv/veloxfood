package repository

import (
	"context"
	"time"

	"project/services/user/internal/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*entity.User, int64, error)

	// GetByIDs returns users matching the given UUIDs. Order is not guaranteed.
	GetByIDs(ctx context.Context, ids []string) ([]*entity.User, error)

	// Auth helpers
	FindByEmailOrPhone(ctx context.Context, identifier string) (*entity.User, error)
	UpdateStatus(ctx context.Context, id string, status entity.UserStatus) error
	// IncrementFailedAttempts atomically increments and returns new count.
	IncrementFailedAttempts(ctx context.Context, id string) (int, error)
	ResetFailedAttempts(ctx context.Context, id string) error
	SetLockedUntil(ctx context.Context, id string, until time.Time) error
	UpdatePassword(ctx context.Context, id, passwordHash string) error
}
