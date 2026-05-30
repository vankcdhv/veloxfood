package http

import (
	authmw "project/pkg/auth/middleware"
	v1 "project/services/user/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles all handler + middleware deps for route registration.
type RouterConfig struct {
	AuthHandler         *v1.AuthHandler
	AuthOAuthHandler    *v1.AuthOAuthHandler
	RoleHandler         *v1.RoleHandler
	PermissionHandler   *v1.PermissionHandler
	UserRoleHandler     *v1.UserRoleHandler
	MeHandler           *v1.MeHandler
	VendorHandler       *v1.VendorHandler
	InvitationHandler   *v1.InvitationHandler
	AdminUserHandler    *v1.AdminUserHandler
	AdminVendorHandler  *v1.AdminVendorHandler
	ShipperHandler      *v1.ShipperHandler
	AdminShipperHandler *v1.AdminShipperHandler
	AuthMiddleware      gin.HandlerFunc
	// PermChecker is the RBAC usecase, used to build per-route PermissionRequired middleware.
	PermChecker authmw.PermissionChecker
}

// RegisterRoutes wires all route groups onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	api := r.Group("/api/v1")

	// ---------- Public: Auth ----------
	if cfg.AuthHandler != nil {
		auth := api.Group("/auth")
		{
			auth.POST("/register", cfg.AuthHandler.Register)
			auth.POST("/verify-register", cfg.AuthHandler.VerifyRegister)
			auth.POST("/login", cfg.AuthHandler.Login)
			auth.POST("/refresh", cfg.AuthHandler.Refresh)
			auth.POST("/logout", cfg.AuthHandler.Logout)
			auth.POST("/change-password", cfg.AuthHandler.ChangePassword)
			auth.POST("/forgot-password", cfg.AuthHandler.ForgotPassword)
			auth.POST("/reset-password", cfg.AuthHandler.ResetPassword)
		}
	}

	// ---------- Public: Google OAuth ----------
	if cfg.AuthOAuthHandler != nil {
		oauthGrp := api.Group("/auth/google")
		oauthGrp.GET("/login", cfg.AuthOAuthHandler.GoogleLogin)
		oauthGrp.GET("/callback", cfg.AuthOAuthHandler.GoogleCallback)
	}

	// ---------- Protected routes — require valid JWT ----------
	if cfg.AuthMiddleware == nil || cfg.PermChecker == nil {
		// Auth or RBAC not configured (dev without secrets) — skip protected routes.
		return
	}

	protected := api.Group("", cfg.AuthMiddleware)
	perm := func(code string) gin.HandlerFunc {
		return authmw.PermissionRequired(cfg.PermChecker, code)
	}

	// Roles
	if cfg.RoleHandler != nil {
		roles := protected.Group("/roles")
		roles.GET("", perm("role.read"), cfg.RoleHandler.List)
		roles.POST("", perm("role.create"), cfg.RoleHandler.Create)
		roles.GET("/:id", perm("role.read"), cfg.RoleHandler.Get)
		roles.PATCH("/:id", perm("role.update"), cfg.RoleHandler.Update)
		roles.DELETE("/:id", perm("role.delete"), cfg.RoleHandler.Delete)
		roles.GET("/:id/permissions", perm("role.read"), cfg.RoleHandler.ListPermissions)
		roles.POST("/:id/permissions/:permission_id", perm("role.assign_permission"), cfg.RoleHandler.AssignPermission)
		roles.DELETE("/:id/permissions/:permission_id", perm("role.assign_permission"), cfg.RoleHandler.RevokePermission)
	}

	// Permissions
	if cfg.PermissionHandler != nil {
		perms := protected.Group("/permissions")
		perms.GET("", perm("permission.read"), cfg.PermissionHandler.List)
		perms.POST("", perm("permission.create"), cfg.PermissionHandler.Create)
		perms.GET("/:id", perm("permission.read"), cfg.PermissionHandler.Get)
		perms.PATCH("/:id", perm("permission.update"), cfg.PermissionHandler.Update)
		perms.DELETE("/:id", perm("permission.delete"), cfg.PermissionHandler.Delete)
	}

	// User roles (non-admin, view own assignments)
	if cfg.UserRoleHandler != nil {
		userRoles := protected.Group("/users/:id/roles")
		userRoles.GET("", perm("user.read"), cfg.UserRoleHandler.ListUserRoles)
	}

	// Self-service profile — AuthRequired only; user acts on own data.
	if cfg.MeHandler != nil {
		me := protected.Group("/me")
		me.GET("", cfg.MeHandler.GetMe)
		me.PATCH("", cfg.MeHandler.UpdateMe)
	}

	// Admin user management.
	if cfg.AdminUserHandler != nil {
		adminUsers := protected.Group("/admin/users")
		adminUsers.GET("", perm("user.read"), cfg.AdminUserHandler.ListUsers)
		adminUsers.POST("/:user_id/suspend", perm("user.suspend"), cfg.AdminUserHandler.SuspendUser)
		adminUsers.POST("/:user_id/reactivate", perm("user.suspend"), cfg.AdminUserHandler.ReactivateUser)
		adminUsers.POST("/:user_id/roles", perm("user.assign_role"), cfg.AdminUserHandler.AssignRole)
		adminUsers.DELETE("/:user_id/roles/:role_id", perm("user.assign_role"), cfg.AdminUserHandler.RemoveRole)
	}

	// Shipper self-registration (any authenticated user applies).
	if cfg.ShipperHandler != nil {
		protected.POST("/shipper/register", cfg.ShipperHandler.Register)
	}

	// Admin shipper approval.
	if cfg.AdminShipperHandler != nil {
		adminShippers := protected.Group("/admin/shippers")
		adminShippers.GET("", perm("shipper.approve"), cfg.AdminShipperHandler.List)
		adminShippers.POST("/:user_id/approve", perm("shipper.approve"), cfg.AdminShipperHandler.Approve)
		adminShippers.POST("/:user_id/reject", perm("shipper.reject"), cfg.AdminShipperHandler.Reject)
	}

	// Admin vendor management.
	if cfg.AdminVendorHandler != nil {
		adminVendors := protected.Group("/admin/vendors")
		adminVendors.POST("/:vendor_id/approve", perm("vendor.approve"), cfg.AdminVendorHandler.ApproveVendor)
		adminVendors.POST("/:vendor_id/reject", perm("vendor.reject"), cfg.AdminVendorHandler.RejectVendor)
	}

	// Vendor onboarding + staff management.
	if cfg.VendorHandler != nil {
		vendorPerm := func(code string) gin.HandlerFunc {
			return authmw.VendorPermissionRequired(cfg.PermChecker, code, "vendor_id")
		}

		protected.POST("/vendors/onboard", cfg.VendorHandler.Onboard)
		protected.GET("/me/vendors", cfg.VendorHandler.MyVendors)

		vendors := protected.Group("/vendors/:vendor_id")
		vendors.GET("/staff", vendorPerm("vendor_member.list"), cfg.VendorHandler.ListStaff)
		vendors.DELETE("/staff/:user_id", vendorPerm("vendor_member.remove"), cfg.VendorHandler.RemoveStaff)
	}

	// Invitation send + accept.
	if cfg.InvitationHandler != nil {
		vendorPerm := func(code string) gin.HandlerFunc {
			return authmw.VendorPermissionRequired(cfg.PermChecker, code, "vendor_id")
		}

		protected.GET("/vendors/:vendor_id/invitations",
			vendorPerm("vendor_member.invite"),
			cfg.InvitationHandler.Invite)
		protected.POST("/vendors/:vendor_id/invitations",
			vendorPerm("vendor_member.invite"),
			cfg.InvitationHandler.Invite)
		protected.POST("/invitations/accept", cfg.InvitationHandler.Accept)
	}
}
