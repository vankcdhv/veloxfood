package usecase

import (
	"context"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
)

// UserUsecase provides read-only user queries used by gRPC and internal services.
// Admin mutation operations are handled by AdminUserUsecase.
type UserUsecase interface {
	GetByID(ctx context.Context, id string) (*entity.User, error)
}

type userUsecase struct {
	userRepo repository.UserRepository
}

func NewUserUsecase(userRepo repository.UserRepository) UserUsecase {
	return &userUsecase{userRepo: userRepo}
}

func (uc *userUsecase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
