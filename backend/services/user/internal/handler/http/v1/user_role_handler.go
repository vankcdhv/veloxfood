package v1

import (
	"net/http"
	"time"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// UserRoleHandler handles role assignment for users.
type UserRoleHandler struct {
	rbacUC usecase.RBACUsecase
}

func NewUserRoleHandler(rbacUC usecase.RBACUsecase) *UserRoleHandler {
	return &UserRoleHandler{rbacUC: rbacUC}
}

type assignRoleRequest struct {
	RoleID    string            `json:"role_id" binding:"required"`
	ScopeType entity.ScopeType  `json:"scope_type" binding:"required"`
	ScopeID   *string           `json:"scope_id"`
	ExpiresAt *time.Time        `json:"expires_at"`
}

func (h *UserRoleHandler) ListUserRoles(c *gin.Context) {
	userID := c.Param("id")
	roles, err := h.rbacUC.ListUserRoles(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, roles)
}

func (h *UserRoleHandler) AssignRoleToUser(c *gin.Context) {
	var req assignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	userID := c.Param("id")

	err := h.rbacUC.AssignRoleToUser(ctx, adminID, userID, req.RoleID, req.ScopeType, req.ScopeID, req.ExpiresAt)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Response{Status: http.StatusCreated, Message: "role assigned"})
}

func (h *UserRoleHandler) RemoveRoleFromUser(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")
	roleID := c.Param("role_id")

	scopeType := entity.ScopeType(c.DefaultQuery("scope_type", string(entity.ScopeTypeGlobal)))
	var scopeID *string
	if sid := c.Query("scope_id"); sid != "" {
		scopeID = &sid
	}

	err := h.rbacUC.RemoveRoleFromUser(ctx, userID, roleID, scopeType, scopeID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
