package main

import (
	"log/slog"

	"project/pkg/app"
	v1 "project/services/user/internal/handler/http/v1"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"
)

// profileHandlers groups self-service profile handlers.
type profileHandlers struct {
	meHandler *v1.MeHandler
}

// buildProfileHandlers wires repositories, usecases, and handlers for profile self-service.
// rbacUC must already be built (from buildRBACHandlers).
func buildProfileHandlers(deps app.Dependencies, rbacUC usecase.RBACUsecase) *profileHandlers {
	userRepo := persistence.NewUserGormRepository(deps.DB)
	membershipRepo := persistence.NewVendorMembershipGormRepository(deps.DB)

	profileUC := usecase.NewProfileUsecase(userRepo, membershipRepo, rbacUC)

	slog.Info("profile handlers wired")
	return &profileHandlers{
		meHandler: v1.NewMeHandler(profileUC),
	}
}
