package v1

import (
	"net/http"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// RoleHandler handles CRUD for roles.
type RoleHandler struct {
	roleUC usecase.RoleUsecase
}

func NewRoleHandler(roleUC usecase.RoleUsecase) *RoleHandler {
	return &RoleHandler{roleUC: roleUC}
}

type createRoleRequest struct {
	Code        string           `json:"code" binding:"required"`
	Name        string           `json:"name" binding:"required"`
	Description *string          `json:"description"`
	ScopeType   entity.ScopeType `json:"scope_type"`
}

type updateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *RoleHandler) List(c *gin.Context) {
	filter := repository.RoleFilter{Limit: 50}
	if st := c.Query("scope_type"); st != "" {
		s := entity.ScopeType(st)
		filter.ScopeType = &s
	}
	if isSys := c.Query("is_system"); isSys == "true" {
		b := true
		filter.IsSystem = &b
	} else if isSys == "false" {
		b := false
		filter.IsSystem = &b
	}

	roles, err := h.roleUC.List(c.Request.Context(), filter)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, roles)
}

func (h *RoleHandler) Get(c *gin.Context) {
	role, err := h.roleUC.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, role)
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	role, err := h.roleUC.Create(c.Request.Context(), usecase.CreateRoleInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		ScopeType:   req.ScopeType,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Response{Status: http.StatusCreated, Message: "created", Data: role})
}

func (h *RoleHandler) Update(c *gin.Context) {
	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	role, err := h.roleUC.Update(c.Request.Context(), c.Param("id"), usecase.UpdateRoleInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, role)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	if err := h.roleUC.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *RoleHandler) ListPermissions(c *gin.Context) {
	perms, err := h.roleUC.ListPermissions(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, perms)
}

func (h *RoleHandler) AssignPermission(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	err := h.roleUC.AssignPermission(ctx, c.Param("id"), c.Param("permission_id"), adminID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *RoleHandler) RevokePermission(c *gin.Context) {
	ctx := c.Request.Context()
	err := h.roleUC.RevokePermission(ctx, c.Param("id"), c.Param("permission_id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
