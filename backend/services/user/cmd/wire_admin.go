package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/app"
	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/pkg/auth/store"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// adminHandlers groups admin-specific HTTP handlers.
type adminHandlers struct {
	adminUserHandler   *v1.AdminUserHandler
	adminVendorHandler *v1.AdminVendorHandler
}

// buildAdminHandlers wires repositories, usecases, and handlers for admin endpoints.
// rbacUC must already be built (from buildRBACHandlers).
func buildAdminHandlers(deps app.Dependencies, rbacUC usecase.RBACUsecase) *adminHandlers {
	cfg := deps.Config

	privKey, err := authjwt.LoadPrivateKey(cfg.JWT.PrivateKeyPath, cfg.JWT.PrivateKeyPEM)
	if err != nil {
		slog.Error("admin wire: JWT key load failed — suspend will not revoke tokens", "err", err)
		return buildAdminHandlersFull(deps, rbacUC, &noopAuthStore{})
	}
	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		slog.Error("admin wire: JWT key load failed — suspend will not revoke tokens", "err", err)
		return buildAdminHandlersFull(deps, rbacUC, &noopAuthStore{})
	}
	_ = authjwt.NewRS256Service(privKey, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	authStore, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		slog.Error("admin wire: auth redis store unavailable — suspend will not revoke tokens", "err", err)
		return buildAdminHandlersFull(deps, rbacUC, &noopAuthStore{})
	}

	return buildAdminHandlersFull(deps, rbacUC, authStore)
}

func buildAdminHandlersFull(deps app.Dependencies, rbacUC usecase.RBACUsecase, authStore store.AuthStore) *adminHandlers {
	userRepo := persistence.NewUserGormRepository(deps.DB)
	studentRepo := persistence.NewStudentProfileGormRepository(deps.DB)
	facultyRepo := persistence.NewFacultyProfileGormRepository(deps.DB)
	membershipRepo := persistence.NewVendorMembershipGormRepository(deps.DB)
	outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
	auditLogger := audit.NewGormLogger(deps.DB)

	adminUserUC := usecase.NewAdminUserUsecase(
		deps.DB,
		userRepo,
		studentRepo,
		facultyRepo,
		outboxRepo,
		rbacUC,
		authStore,
		auditLogger,
	)

	adminVendorUC := usecase.NewAdminVendorUsecase(
		deps.DB,
		membershipRepo,
		outboxRepo,
		rbacUC,
		auditLogger,
	)

	slog.Info("admin handlers wired")
	return &adminHandlers{
		adminUserHandler:   v1.NewAdminUserHandler(adminUserUC),
		adminVendorHandler: v1.NewAdminVendorHandler(adminVendorUC),
	}
}

// noopAuthStore is a fail-safe placeholder when Redis is unavailable.
// All operations return errors — Suspend logs but does not fail.
type noopAuthStore struct{}

func (n *noopAuthStore) Whitelist(_ context.Context, _, _ string, _ time.Duration) error {
	return fmt.Errorf("auth store unavailable")
}
func (n *noopAuthStore) Exists(_ context.Context, _ string) (bool, error) {
	return false, fmt.Errorf("auth store unavailable")
}
func (n *noopAuthStore) Revoke(_ context.Context, _, _ string) error {
	return fmt.Errorf("auth store unavailable")
}
func (n *noopAuthStore) RevokeAll(_ context.Context, _ string) error {
	return fmt.Errorf("auth store unavailable — token revocation skipped")
}
func (n *noopAuthStore) RotatePair(
	_ context.Context,
	_, _, _, _, _ string,
	_, _ time.Duration,
) error {
	return fmt.Errorf("auth store unavailable")
}
