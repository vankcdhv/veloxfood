package usecase

import "project/pkg/apperror"

var (
	ErrInvalidCredentials = apperror.Unauthorized("invalid credentials")
	ErrAccountLocked      = apperror.Unauthorized("account locked, try again later")
	ErrAccountPending     = apperror.Unauthorized("account not verified")
	ErrAccountSuspended   = apperror.Unauthorized("account suspended")
	ErrOTPInvalid         = apperror.BadRequest("invalid or expired OTP code")
	ErrOTPExpired         = apperror.BadRequest("OTP code has expired")
	ErrOTPMaxAttempts     = apperror.BadRequest("too many OTP attempts")
	ErrRefreshLeaked      = apperror.Unauthorized("session invalidated, please login again")
	ErrTokenInvalid       = apperror.Unauthorized("invalid token")
	ErrResetTokenInvalid  = apperror.BadRequest("invalid or expired reset token")
	ErrIdentifierRequired = apperror.BadRequest("email or phone is required")
	ErrPasswordRequired   = apperror.BadRequest("password is required")
	ErrWeakPassword       = apperror.BadRequest("password must be at least 8 characters")
)
