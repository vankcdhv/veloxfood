package repository

import (
	"context"

	"project/services/user/internal/entity"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, token *entity.PasswordResetToken) error
	FindByTokenHash(ctx context.Context, hash string) (*entity.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
}
