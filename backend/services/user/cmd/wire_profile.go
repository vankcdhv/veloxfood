package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// profileHandlers groups profile + card handlers.
type profileHandlers struct {
	meHandler        *v1.MeHandler
	cardAdminHandler *v1.CardAdminHandler
}

// buildProfileHandlers wires repositories, usecases, and handlers for profile self-service and card admin.
// rbacUC must already be built (from buildRBACHandlers).
func buildProfileHandlers(deps app.Dependencies, rbacUC usecase.RBACUsecase) *profileHandlers {
	userRepo := persistence.NewUserGormRepository(deps.DB)
	studentRepo := persistence.NewStudentProfileGormRepository(deps.DB)
	facultyRepo := persistence.NewFacultyProfileGormRepository(deps.DB)
	cardRepo := persistence.NewCardGormRepository(deps.DB)
	membershipRepo := persistence.NewVendorMembershipGormRepository(deps.DB)

	auditLogger := audit.NewGormLogger(deps.DB)
	profileUC := usecase.NewProfileUsecase(userRepo, studentRepo, facultyRepo, membershipRepo, rbacUC)
	cardUC := usecase.NewCardUsecase(cardRepo, userRepo, auditLogger)

	slog.Info("profile + card handlers wired")
	return &profileHandlers{
		meHandler:        v1.NewMeHandler(profileUC),
		cardAdminHandler: v1.NewCardAdminHandler(cardUC),
	}
}
