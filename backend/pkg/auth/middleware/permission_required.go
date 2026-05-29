package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"project/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PermissionChecker abstracts the RBAC usecase to avoid import cycles.
// Implemented by usecase.RBACUsecase.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, code string) (bool, error)
	HasVendorPermission(ctx context.Context, userID, vendorID, code string) (bool, error)
}

// PermissionRequired checks that the authenticated user has the given global permission.
// Must run after AuthRequired (requires user_id in context).
func PermissionRequired(checker PermissionChecker, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := UserIDFromContext(ctx)
		if userID == "" {
			slog.WarnContext(ctx, "permission_required: no user_id in context — AuthRequired not applied")
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
				Error:   "missing authentication",
			})
			return
		}

		ok, err := checker.HasPermission(ctx, userID, code)
		if err != nil {
			slog.ErrorContext(ctx, "permission_required: check failed", "code", code, "err", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
				Status:  http.StatusInternalServerError,
				Message: "internal server error",
				Error:   "permission check failed",
			})
			return
		}
		if !ok {
			slog.DebugContext(ctx, "permission_required: denied", "user_id", userID, "code", code)
			c.AbortWithStatusJSON(http.StatusForbidden, response.Response{
				Status:  http.StatusForbidden,
				Message: "forbidden",
				Error:   "insufficient permissions",
			})
			return
		}

		c.Next()
	}
}

// VendorPermissionRequired checks that the authenticated user has the given permission
// scoped to the vendor ID extracted from the named route parameter.
// Must run after AuthRequired.
func VendorPermissionRequired(checker PermissionChecker, code, vendorIDParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userID := UserIDFromContext(ctx)
		if userID == "" {
			slog.WarnContext(ctx, "vendor_permission_required: no user_id in context")
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
				Error:   "missing authentication",
			})
			return
		}

		vendorID := c.Param(vendorIDParam)
		if _, err := uuid.Parse(vendorID); err != nil {
			slog.DebugContext(ctx, "vendor_permission_required: invalid vendor_id", "param", vendorIDParam, "value", vendorID)
			c.AbortWithStatusJSON(http.StatusBadRequest, response.Response{
				Status:  http.StatusBadRequest,
				Message: "bad request",
				Error:   "invalid vendor_id — must be a UUID",
			})
			return
		}

		ok, err := checker.HasVendorPermission(ctx, userID, vendorID, code)
		if err != nil {
			slog.ErrorContext(ctx, "vendor_permission_required: check failed", "code", code, "vendor_id", vendorID, "err", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Response{
				Status:  http.StatusInternalServerError,
				Message: "internal server error",
				Error:   "permission check failed",
			})
			return
		}
		if !ok {
			slog.DebugContext(ctx, "vendor_permission_required: denied", "user_id", userID, "vendor_id", vendorID, "code", code)
			c.AbortWithStatusJSON(http.StatusForbidden, response.Response{
				Status:  http.StatusForbidden,
				Message: "forbidden",
				Error:   "insufficient permissions for this vendor",
			})
			return
		}

		c.Next()
	}
}
