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

			// Profile handlers depend on RBAC usecase for ListUserRoles.
			profile := buildProfileHandlers(deps, rbac.rbacUC)
			cfg.MeHandler = profile.meHandler

			// Google OAuth (optional — disabled if not configured).
			if oauthHandler, oerr := buildOAuthHandler(deps, rbac.rbacUC); oerr != nil {
				slog.Error("oauth handler setup failed — google login disabled", "err", oerr)
			} else {
				cfg.AuthOAuthHandler = oauthHandler
			}

			// Vendor onboarding + staff management.
			vendor := buildVendorHandlers(deps, rbac.rbacUC)
			cfg.VendorHandler = vendor.vendorHandler
			cfg.InvitationHandler = vendor.invitationHandler

			// Admin endpoints.
			admin := buildAdminHandlers(deps, rbac.rbacUC)
			cfg.AdminUserHandler = admin.adminUserHandler
			cfg.AdminVendorHandler = admin.adminVendorHandler

			// Shipper registration + admin approval (MinIO-backed uploads).
			uploader := newMinioUploader(deps)
			shipper := buildShipperHandlers(deps, rbac.rbacUC, uploader)
			cfg.ShipperHandler = shipper.shipperHandler
			cfg.AdminShipperHandler = shipper.adminShipperHandler
		}

		handlerhttp.RegisterRoutes(r, cfg)

		// Dispatcher needs deps ready; piggy-back on the HTTP registrar callback
		// which runs after DB + Cache are initialised.
		startOutboxDispatcher(a, deps)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		userRepo := persistence.NewUserGormRepository(deps.DB)
		membershipRepo := persistence.NewVendorMembershipGormRepository(deps.DB)
		rbacRepo := persistence.NewRBACGormRepository(deps.DB)
		userUC := usecase.NewUserUsecase(userRepo)
		rbacUC := usecase.NewRBACUsecase(rbacRepo, deps.Cache)
		userv1.RegisterUserServiceServer(s, grpchandler.NewUserServiceServer(userUC, userRepo, membershipRepo, rbacUC))
	})

	a.Run()
}
