package repository

import (
	"context"

	"project/services/user/internal/entity"
)

type OTPRepository interface {
	Create(ctx context.Context, otp *entity.OTPCode) error
	FindActive(ctx context.Context, destination string, purpose entity.OTPPurpose) (*entity.OTPCode, error)
	IncrementAttempts(ctx context.Context, id string) error
	MarkUsed(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
}
