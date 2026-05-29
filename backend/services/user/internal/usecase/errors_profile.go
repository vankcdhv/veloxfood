package usecase

import "project/pkg/apperror"

var (
	ErrProfileNotFound      = apperror.NotFound("profile not found")
	ErrProfileUpdateFailed  = apperror.Internal("failed to update profile")
	ErrAllergiesLimit       = apperror.BadRequest("allergies must not exceed 10 elements")
	ErrInvalidGender        = apperror.BadRequest("invalid gender value")
	ErrFullNameRequired     = apperror.BadRequest("full_name must not be empty")
	ErrUpdateUser           = apperror.Internal("failed to update user")
)
