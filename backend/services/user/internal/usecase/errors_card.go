package usecase

import "project/pkg/apperror"

var (
	ErrCardConflict    = apperror.Conflict("an active card with this identifier already exists")
	ErrCardNotFound    = apperror.NotFound("card not found")
	ErrCardAlreadyRevoked = apperror.New(422, "card is already revoked")
	ErrCardCreateFailed = apperror.Internal("failed to create card")
)
