package v1

import (
	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/location/internal/usecase"

	"github.com/gin-gonic/gin"
)

// MeLocationHandler manages the current customer's saved delivery locations.
type MeLocationHandler struct {
	uc usecase.CustomerLocationUsecase
}

func NewMeLocationHandler(uc usecase.CustomerLocationUsecase) *MeLocationHandler {
	return &MeLocationHandler{uc: uc}
}

func (h *MeLocationHandler) List(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	items, err := h.uc.List(c.Request.Context(), userID)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

type addLocationRequest struct {
	RoomID    string `json:"room_id"`
	Label     string `json:"label"`
	IsDefault bool   `json:"is_default"`
}

func (h *MeLocationHandler) Add(c *gin.Context) {
	var req addLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := authmw.UserIDFromContext(c.Request.Context())
	loc, err := h.uc.Add(c.Request.Context(), userID, req.RoomID, req.Label, req.IsDefault)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, loc)
}

func (h *MeLocationHandler) Remove(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	if err := h.uc.Remove(c.Request.Context(), userID, c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *MeLocationHandler) SetDefault(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	if err := h.uc.SetDefault(c.Request.Context(), userID, c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
