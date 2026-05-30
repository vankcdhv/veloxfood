package usecase

import "project/pkg/apperror"

var (
	ErrRoleNotFound                 = apperror.NotFound("role not found")
	ErrRoleAlreadyExists            = apperror.Conflict("role code already exists")
	ErrCannotDeleteSystemRole       = apperror.New(422, "cannot delete a system role")
	ErrPermissionNotFound           = apperror.NotFound("permission not found")
	ErrPermissionAlreadyExists      = apperror.Conflict("permission code already exists")
	ErrCannotDeleteSystemPermission = apperror.New(422, "cannot delete a system permission")
	ErrUserRoleConflict             = apperror.Conflict("user already has this role in the given scope")
)
