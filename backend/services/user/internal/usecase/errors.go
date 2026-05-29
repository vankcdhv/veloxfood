package usecase

import "project/pkg/apperror"

var (
	ErrUserNotFound = apperror.NotFound("user not found")
	ErrListUsers    = apperror.Internal("failed to list users")
)
