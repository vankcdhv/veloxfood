package v1

import (
	"net/http"

	"project/pkg/response"
	"project/services/user/internal/repository"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// PermissionHandler handles CRUD for permissions.
type PermissionHandler struct {
	permUC usecase.PermissionUsecase
}

func NewPermissionHandler(permUC usecase.PermissionUsecase) *PermissionHandler {
	return &PermissionHandler{permUC: permUC}
}

type createPermissionRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Resource    string  `json:"resource" binding:"required"`
	Action      string  `json:"action" binding:"required"`
}

type updatePermissionRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *PermissionHandler) List(c *gin.Context) {
	filter := repository.PermissionFilter{Limit: 100}
	if r := c.Query("resource"); r != "" {
		filter.Resource = &r
	}
	if a := c.Query("action"); a != "" {
		filter.Action = &a
	}

	perms, err := h.permUC.List(c.Request.Context(), filter)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, perms)
}

func (h *PermissionHandler) Get(c *gin.Context) {
	perm, err := h.permUC.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, perm)
}

func (h *PermissionHandler) Create(c *gin.Context) {
	var req createPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	perm, err := h.permUC.Create(c.Request.Context(), usecase.CreatePermissionInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Response{Status: http.StatusCreated, Message: "created", Data: perm})
}

func (h *PermissionHandler) Update(c *gin.Context) {
	var req updatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	perm, err := h.permUC.Update(c.Request.Context(), c.Param("id"), usecase.UpdatePermissionInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, perm)
}

func (h *PermissionHandler) Delete(c *gin.Context) {
	if err := h.permUC.Delete(c.Request.Context(), c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
