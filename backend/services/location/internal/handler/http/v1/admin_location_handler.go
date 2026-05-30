package v1

import (
	"project/pkg/response"
	"project/services/location/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminLocationHandler manages the Building → Floor → Room tree (perm: location.manage).
type AdminLocationHandler struct {
	uc usecase.LocationUsecase
}

func NewAdminLocationHandler(uc usecase.LocationUsecase) *AdminLocationHandler {
	return &AdminLocationHandler{uc: uc}
}

// ---- Buildings ----

type buildingRequest struct {
	Name     string `json:"name"`
	Address  string `json:"address"`
	IsActive *bool  `json:"is_active"`
}

func (h *AdminLocationHandler) ListBuildings(c *gin.Context) {
	items, err := h.uc.ListBuildings(c.Request.Context(), false)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *AdminLocationHandler) CreateBuilding(c *gin.Context) {
	var req buildingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	b, err := h.uc.CreateBuilding(c.Request.Context(), req.Name, req.Address)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, b)
}

func (h *AdminLocationHandler) UpdateBuilding(c *gin.Context) {
	var req buildingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	active := req.IsActive == nil || *req.IsActive
	b, err := h.uc.UpdateBuilding(c.Request.Context(), c.Param("id"), req.Name, req.Address, active)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, b)
}

func (h *AdminLocationHandler) DeleteBuilding(c *gin.Context) {
	if err := h.uc.DeleteBuilding(c.Request.Context(), c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// ---- Floors ----

type floorRequest struct {
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

func (h *AdminLocationHandler) ListFloors(c *gin.Context) {
	items, err := h.uc.ListFloors(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *AdminLocationHandler) CreateFloor(c *gin.Context) {
	var req floorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	f, err := h.uc.CreateFloor(c.Request.Context(), c.Param("id"), req.Name, req.SortOrder)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, f)
}

func (h *AdminLocationHandler) UpdateFloor(c *gin.Context) {
	var req floorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	f, err := h.uc.UpdateFloor(c.Request.Context(), c.Param("id"), req.Name, req.SortOrder)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, f)
}

func (h *AdminLocationHandler) DeleteFloor(c *gin.Context) {
	if err := h.uc.DeleteFloor(c.Request.Context(), c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// ---- Rooms ----

type roomRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}

func (h *AdminLocationHandler) ListRooms(c *gin.Context) {
	items, err := h.uc.ListRooms(c.Request.Context(), c.Param("id"), false)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *AdminLocationHandler) CreateRoom(c *gin.Context) {
	var req roomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	r, err := h.uc.CreateRoom(c.Request.Context(), c.Param("id"), req.Code, req.Name)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, r)
}

func (h *AdminLocationHandler) UpdateRoom(c *gin.Context) {
	var req roomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	active := req.IsActive == nil || *req.IsActive
	r, err := h.uc.UpdateRoom(c.Request.Context(), c.Param("id"), req.Code, req.Name, active)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, r)
}

func (h *AdminLocationHandler) DeleteRoom(c *gin.Context) {
	if err := h.uc.DeleteRoom(c.Request.Context(), c.Param("id")); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
