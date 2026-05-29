package usecase

import (
	"time"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/pkg/auth/store"
	"project/pkg/mailer"
	"project/services/user/internal/repository"
)

const (
	maxLoginAttempts = 5
	lockoutDuration  = 15 * time.Minute
)

// authUsecase is the shared receiver for all auth sub-usecases.
type authUsecase struct {
	userRepo    repository.UserRepository
	otpRepo     repository.OTPRepository
	resetRepo   repository.PasswordResetRepository
	jwtSvc      authjwt.JWTService
	store       store.AuthStore
	mailer      mailer.Mailer
	otpSvc      OTPService
	auditLogger audit.Logger
}

// AuthUsecase composes all auth operations.
type AuthUsecase interface {
	AuthRegister
	AuthLogin
	AuthRefresh
	AuthLogout
	AuthPassword
}

// NewAuthUsecase constructs the unified auth usecase.
func NewAuthUsecase(
	userRepo repository.UserRepository,
	otpRepo repository.OTPRepository,
	resetRepo repository.PasswordResetRepository,
	jwtSvc authjwt.JWTService,
	authStore store.AuthStore,
	m mailer.Mailer,
	otpSvc OTPService,
	auditLogger audit.Logger,
) AuthUsecase {
	return &authUsecase{
		userRepo:    userRepo,
		otpRepo:     otpRepo,
		resetRepo:   resetRepo,
		jwtSvc:      jwtSvc,
		store:       authStore,
		mailer:      m,
		otpSvc:      otpSvc,
		auditLogger: auditLogger,
	}
}
