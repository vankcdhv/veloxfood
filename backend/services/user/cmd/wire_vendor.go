package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	"project/pkg/mailer"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// vendorHandlers groups vendor + invitation handlers.
type vendorHandlers struct {
	vendorHandler     *v1.VendorHandler
	invitationHandler *v1.InvitationHandler
}

// buildVendorHandlers wires repositories, usecases, and handlers for vendor operations.
// rbacUC must already be built (from buildRBACHandlers).
func buildVendorHandlers(deps app.Dependencies, rbacUC usecase.RBACUsecase) *vendorHandlers {
	membershipRepo := persistence.NewVendorMembershipGormRepository(deps.DB)
	invitationRepo := persistence.NewInvitationGormRepository(deps.DB)
	outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
	roleRepo := persistence.NewRoleGormRepository(deps.DB)

	m := mailer.NewSMTPMailer(deps.Config.SMTP)

	baseURL := deps.Config.App.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	auditLogger := audit.NewGormLogger(deps.DB)
	onboardUC := usecase.NewVendorOnboardUsecase(deps.DB, membershipRepo, roleRepo, outboxRepo, rbacUC, auditLogger)
	staffUC := usecase.NewVendorStaffUsecase(deps.DB, invitationRepo, membershipRepo, roleRepo, outboxRepo, rbacUC, m, baseURL, auditLogger)

	slog.Info("vendor handlers wired")
	return &vendorHandlers{
		vendorHandler:     v1.NewVendorHandler(onboardUC, staffUC),
		invitationHandler: v1.NewInvitationHandler(staffUC),
	}
}
