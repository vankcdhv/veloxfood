package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/middleware"
	userv1 "project/proto/user/v1"
	grpchandler "project/services/user/internal/handler/grpc"
	handlerhttp "project/services/user/internal/handler/http"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("user-service")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		// RequestMetadata injects client IP + User-Agent into context for audit logging.
		r.Use(middleware.RequestMetadata())

		cfg := handlerhttp.RouterConfig{}

		authHandler, err := buildAuthHandler(deps)
		if err != nil {
			slog.Error("auth handler setup failed — auth routes disabled", "err", err)
		} else {
			cfg.AuthHandler = authHandler
		}

		rbac, err := buildRBACHandlers(deps)
		if err != nil {
			slog.Error("RBAC handlers setup failed — RBAC routes disabled", "err", err)
		} else {
			cfg.RoleHandler = rbac.roleHandler
			cfg.PermissionHandler = rbac.permissionHandler
			cfg.UserRoleHandler = rbac.userRoleHandler
			cfg.AuthMiddleware = rbac.authMiddleware
			cfg.PermChecker = rbac.permChecker

			// Profile + card handlers depend on RBAC usecase for ListUserRoles.
			profile := buildProfileHandlers(deps, rbac.rbacUC)
			cfg.MeHandler = profile.meHandler
			cfg.CardAdminHandler = profile.cardAdminHandler

			// Vendor onboarding + staff management.
			vendor := buildVendorHandlers(deps, rbac.rbacUC)
			cfg.VendorHandler = vendor.vendorHandler
			cfg.InvitationHandler = vendor.invitationHandler

			// Admin endpoints.
			admin := buildAdminHandlers(deps, rbac.rbacUC)
			cfg.AdminUserHandler = admin.adminUserHandler
			cfg.AdminVendorHandler = admin.adminVendorHandler
		}

		handlerhttp.RegisterRoutes(r, cfg)

		// Dispatcher needs deps ready; piggy-back on the HTTP registrar callback
		// which runs after DB + Cache are initialised.
		startOutboxDispatcher(a, deps)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		userRepo := persistence.NewUserGormRepository(deps.DB)
		membershipRepo := persistence.NewVendorMembershipGormRepository(deps.DB)
		userUC := usecase.NewUserUsecase(userRepo)
		userv1.RegisterUserServiceServer(s, grpchandler.NewUserServiceServer(userUC, userRepo, membershipRepo))
	})

	a.Run()
}
