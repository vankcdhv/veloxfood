package main

import (
	"fmt"
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/pkg/auth/store"
	"project/pkg/mailer"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// buildAuthHandler wires auth dependencies from app config + DB.
// Returns nil if RSA keys are not configured (dev convenience — logs warning).
func buildAuthHandler(deps app.Dependencies) (*v1.AuthHandler, error) {
	cfg := deps.Config

	privKey, err := authjwt.LoadPrivateKey(cfg.JWT.PrivateKeyPath, cfg.JWT.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}
	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("load public key: %w", err)
	}

	jwtSvc := authjwt.NewRS256Service(privKey, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	authStore, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("auth redis store: %w", err)
	}

	m := mailer.NewSMTPMailer(cfg.SMTP)

	userRepo := persistence.NewUserGormRepository(deps.DB)
	otpRepo := persistence.NewOTPGormRepository(deps.DB)
	resetRepo := persistence.NewPasswordResetGormRepository(deps.DB)

	otpSvc := usecase.NewOTPService(otpRepo, m, cfg.OTP.TTL, cfg.OTP.MaxAttempts, cfg.App.Env)

	auditLogger := audit.NewGormLogger(deps.DB)
	authUC := usecase.NewAuthUsecase(userRepo, otpRepo, resetRepo, jwtSvc, authStore, m, otpSvc, auditLogger)

	slog.Info("auth handler wired")
	return v1.NewAuthHandler(authUC, v1.CookieConfig{
		AccessTTL:  cfg.JWT.AccessTTL,
		RefreshTTL: cfg.JWT.RefreshTTL,
		Secure:     cfg.App.CookieSecure,
		Domain:     cfg.App.CookieDomain,
	}), nil
}
