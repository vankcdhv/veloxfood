package usecase

import (
	"context"
	"errors"

	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"github.com/google/uuid"
)

// ErrInvalidFavoriteTarget rejects malformed type/id pairs before they hit SQL.
var ErrInvalidFavoriteTarget = errors.New("invalid favorite target")

// FavoriteUsecase manages a customer's bookmarked stores and menu items.
type FavoriteUsecase interface {
	Add(ctx context.Context, userID, targetType, targetID string) error
	Remove(ctx context.Context, userID, targetType, targetID string) error
	IDs(ctx context.Context, userID, targetType string) ([]string, error)
	Stores(ctx context.Context, userID string) ([]*entity.Store, error)
	Items(ctx context.Context, userID string) ([]repository.SearchMenuItemRow, error)
}

type favoriteUsecase struct {
	repo repository.FavoriteRepository
}

func NewFavoriteUsecase(repo repository.FavoriteRepository) FavoriteUsecase {
	return &favoriteUsecase{repo: repo}
}

// parseTarget maps the URL segment ("store"/"item", case-insensitive) to the
// entity enum and validates the id is a UUID.
func parseTarget(targetType, targetID string) (entity.FavoriteTargetType, error) {
	if _, err := uuid.Parse(targetID); err != nil {
		return "", ErrInvalidFavoriteTarget
	}
	switch targetType {
	case "store", "STORE":
		return entity.FavoriteTargetStore, nil
	case "item", "ITEM":
		return entity.FavoriteTargetItem, nil
	}
	return "", ErrInvalidFavoriteTarget
}

func (uc *favoriteUsecase) Add(ctx context.Context, userID, targetType, targetID string) error {
	tt, err := parseTarget(targetType, targetID)
	if err != nil {
		return err
	}
	return uc.repo.Add(ctx, &entity.Favorite{UserID: userID, TargetType: tt, TargetID: targetID})
}

func (uc *favoriteUsecase) Remove(ctx context.Context, userID, targetType, targetID string) error {
	tt, err := parseTarget(targetType, targetID)
	if err != nil {
		return err
	}
	return uc.repo.Remove(ctx, userID, tt, targetID)
}

func (uc *favoriteUsecase) IDs(ctx context.Context, userID, targetType string) ([]string, error) {
	// IDs listing needs no target id — validate the type alone.
	switch targetType {
	case "store", "STORE":
		return uc.repo.IDs(ctx, userID, entity.FavoriteTargetStore)
	case "item", "ITEM":
		return uc.repo.IDs(ctx, userID, entity.FavoriteTargetItem)
	}
	return nil, ErrInvalidFavoriteTarget
}

func (uc *favoriteUsecase) Stores(ctx context.Context, userID string) ([]*entity.Store, error) {
	return uc.repo.FavoriteStores(ctx, userID)
}

func (uc *favoriteUsecase) Items(ctx context.Context, userID string) ([]repository.SearchMenuItemRow, error) {
	return uc.repo.FavoriteItems(ctx, userID)
}
