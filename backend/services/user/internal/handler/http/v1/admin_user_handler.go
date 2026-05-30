package v1

import (
	"net/http"
	"strconv"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminUserHandler handles admin-initiated user management endpoints.
type AdminUserHandler struct {
	adminUserUC usecase.AdminUserUsecase
}

func NewAdminUserHandler(adminUserUC usecase.AdminUserUsecase) *AdminUserHandler {
	return &AdminUserHandler{adminUserUC: adminUserUC}
}

// SuspendUser POST /admin/users/:user_id/suspend
func (h *AdminUserHandler) SuspendUser(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	userID := c.Param("user_id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.adminUserUC.Suspend(ctx, adminID, userID, req.Reason); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// ReactivateUser POST /admin/users/:user_id/reactivate
func (h *AdminUserHandler) ReactivateUser(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	userID := c.Param("user_id")

	if err := h.adminUserUC.Reactivate(ctx, adminID, userID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// ListUsers GET /admin/users
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := usecase.ListUsersFilter{
		RoleID:   c.Query("role_id"),
		VendorID: c.Query("vendor_id"),
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		Page:     page,
		PageSize: pageSize,
	}

	users, total, err := h.adminUserUC.ListUsers(c.Request.Context(), filter)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Paginated(c, toUserListResponse(users), total, page)
}

// AssignRole POST /admin/users/:user_id/roles
func (h *AdminUserHandler) AssignRole(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	userID := c.Param("user_id")

	var req struct {
		RoleID    string           `json:"role_id" binding:"required"`
		ScopeType entity.ScopeType `json:"scope_type" binding:"required"`
		ScopeID   *string          `json:"scope_id"`
		ExpiresAt *time.Time       `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.adminUserUC.AssignRole(ctx, adminID, userID, req.RoleID, req.ScopeType, req.ScopeID, req.ExpiresAt); err != nil {
		response.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Response{Status: http.StatusCreated, Message: "role assigned"})
}

// RemoveRole DELETE /admin/users/:user_id/roles/:role_id
func (h *AdminUserHandler) RemoveRole(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	userID := c.Param("user_id")
	roleID := c.Param("role_id")

	scopeType := entity.ScopeType(c.DefaultQuery("scope_type", string(entity.ScopeTypeGlobal)))
	var scopeID *string
	if sid := c.Query("scope_id"); sid != "" {
		scopeID = &sid
	}

	if err := h.adminUserUC.RemoveRole(ctx, adminID, userID, roleID, scopeType, scopeID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
