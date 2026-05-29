package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// PermissionUsecase manages permission CRUD operations.
type PermissionUsecase interface {
	List(ctx context.Context, filter repository.PermissionFilter) ([]*entity.Permission, error)
	GetByID(ctx context.Context, id string) (*entity.Permission, error)
	Create(ctx context.Context, in CreatePermissionInput) (*entity.Permission, error)
	Update(ctx context.Context, id string, in UpdatePermissionInput) (*entity.Permission, error)
	Delete(ctx context.Context, id string) error
}

// CreatePermissionInput is the validated payload for creating a permission.
type CreatePermissionInput struct {
	Code        string
	Name        string
	Description *string
	Resource    string
	Action      string
}

// UpdatePermissionInput is the validated payload for patching a permission.
type UpdatePermissionInput struct {
	Name        *string
	Description *string
}

type permissionUsecase struct {
	permRepo repository.PermissionRepository
}

// NewPermissionUsecase constructs PermissionUsecase.
func NewPermissionUsecase(permRepo repository.PermissionRepository) PermissionUsecase {
	return &permissionUsecase{permRepo: permRepo}
}

func (u *permissionUsecase) List(ctx context.Context, filter repository.PermissionFilter) ([]*entity.Permission, error) {
	perms, err := u.permRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return perms, nil
}

func (u *permissionUsecase) GetByID(ctx context.Context, id string) (*entity.Permission, error) {
	perm, err := u.permRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPermissionNotFound
		}
		return nil, fmt.Errorf("get permission: %w", err)
	}
	return perm, nil
}

func (u *permissionUsecase) Create(ctx context.Context, in CreatePermissionInput) (*entity.Permission, error) {
	if _, err := u.permRepo.GetByCode(ctx, in.Code); err == nil {
		return nil, ErrPermissionAlreadyExists
	}

	perm := &entity.Permission{
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		Resource:    in.Resource,
		Action:      in.Action,
		IsSystem:    false,
	}
	if err := u.permRepo.Create(ctx, perm); err != nil {
		return nil, fmt.Errorf("create permission: %w", err)
	}
	slog.InfoContext(ctx, "permission created", "permission_id", perm.ID, "code", perm.Code)
	return perm, nil
}

func (u *permissionUsecase) Update(ctx context.Context, id string, in UpdatePermissionInput) (*entity.Permission, error) {
	perm, err := u.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		perm.Name = *in.Name
	}
	if in.Description != nil {
		perm.Description = in.Description
	}

	if err := u.permRepo.Update(ctx, perm); err != nil {
		return nil, fmt.Errorf("update permission: %w", err)
	}
	return perm, nil
}

func (u *permissionUsecase) Delete(ctx context.Context, id string) error {
	perm, err := u.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if perm.IsSystem {
		return ErrCannotDeleteSystemPermission
	}
	if err := u.permRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete permission: %w", err)
	}
	slog.InfoContext(ctx, "permission deleted", "permission_id", id)
	return nil
}
