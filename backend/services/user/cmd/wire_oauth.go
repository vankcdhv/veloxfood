package main

import (
	"fmt"
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/pkg/auth/store"
	"project/pkg/oauth"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// buildOAuthHandler wires the Google OAuth login flow.
// Returns nil (no error) when Google OAuth is not configured — routes stay disabled.
func buildOAuthHandler(deps app.Dependencies, rbacUC usecase.RBACUsecase) (*v1.AuthOAuthHandler, error) {
	cfg := deps.Config

	client := oauth.NewClient(cfg.GoogleOAuth)
	if client == nil {
		slog.Warn("google oauth not configured — social login disabled")
		return nil, nil
	}

	privKey, err := authjwt.LoadPrivateKey(cfg.JWT.PrivateKeyPath, cfg.JWT.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("oauth wire: load private key: %w", err)
	}
	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("oauth wire: load public key: %w", err)
	}
	jwtSvc := authjwt.NewRS256Service(privKey, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	authStore, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("oauth wire: auth redis store: %w", err)
	}

	userRepo := persistence.NewUserGormRepository(deps.DB)
	identityRepo := persistence.NewIdentityGormRepository(deps.DB)
	roleRepo := persistence.NewRoleGormRepository(deps.DB)
	auditLogger := audit.NewGormLogger(deps.DB)

	oauthUC := usecase.NewOAuthUsecase(deps.DB, userRepo, identityRepo, roleRepo, rbacUC, jwtSvc, authStore, auditLogger)

	webURL := "http://localhost:3000"

	slog.Info("oauth handler wired")
	return v1.NewAuthOAuthHandler(client, oauthUC, v1.CookieConfig{
		AccessTTL:  cfg.JWT.AccessTTL,
		RefreshTTL: cfg.JWT.RefreshTTL,
		Secure:     cfg.App.CookieSecure,
		Domain:     cfg.App.CookieDomain,
	}, webURL), nil
}
