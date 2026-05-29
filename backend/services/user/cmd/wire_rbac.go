package main

import (
	"fmt"
	"log/slog"

	"project/pkg/app"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	"project/pkg/auth/store"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// rbacHandlers groups all RBAC-related HTTP handlers.
type rbacHandlers struct {
	roleHandler       *v1.RoleHandler
	permissionHandler *v1.PermissionHandler
	userRoleHandler   *v1.UserRoleHandler
	authMiddleware    gin.HandlerFunc
	permChecker       authmw.PermissionChecker
	// rbacUC is the full RBAC usecase — exposed so other wire builders can use ListUserRoles etc.
	rbacUC usecase.RBACUsecase
}

// buildRBACHandlers wires repositories, usecases, and handlers for RBAC.
// Returns nil if JWT keys are not configured (graceful degradation).
func buildRBACHandlers(deps app.Dependencies) (*rbacHandlers, error) {
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

	if deps.Cache == nil {
		return nil, fmt.Errorf("cache not available — RBAC requires Redis cache")
	}

	// Repositories
	roleRepo := persistence.NewRoleGormRepository(deps.DB)
	permRepo := persistence.NewPermissionGormRepository(deps.DB)
	rbacRepo := persistence.NewRBACGormRepository(deps.DB)

	// Usecases
	rbacUC := usecase.NewRBACUsecase(rbacRepo, deps.Cache)
	roleUC := usecase.NewRoleUsecase(roleRepo, permRepo, rbacUC)
	permUC := usecase.NewPermissionUsecase(permRepo)

	// Middleware
	authMiddleware := authmw.AuthRequired(jwtSvc, authStore)

	slog.Info("RBAC handlers wired")
	return &rbacHandlers{
		roleHandler:       v1.NewRoleHandler(roleUC),
		permissionHandler: v1.NewPermissionHandler(permUC),
		userRoleHandler:   v1.NewUserRoleHandler(rbacUC),
		authMiddleware:    authMiddleware,
		permChecker:       rbacUC,
		rbacUC:            rbacUC,
	}, nil
}
