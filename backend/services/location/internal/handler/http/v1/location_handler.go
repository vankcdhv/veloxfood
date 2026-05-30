package v1

import (
	"project/pkg/response"
	"project/services/location/internal/usecase"

	"github.com/gin-gonic/gin"
)

// LocationHandler exposes the read-only location tree for customers to pick from.
type LocationHandler struct {
	uc usecase.LocationUsecase
}

func NewLocationHandler(uc usecase.LocationUsecase) *LocationHandler {
	return &LocationHandler{uc: uc}
}

// ListBuildings GET /api/v1/locations/buildings — active buildings only.
func (h *LocationHandler) ListBuildings(c *gin.Context) {
	items, err := h.uc.ListBuildings(c.Request.Context(), true)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// ListFloors GET /api/v1/locations/buildings/:id/floors.
func (h *LocationHandler) ListFloors(c *gin.Context) {
	items, err := h.uc.ListFloors(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// ListRooms GET /api/v1/locations/floors/:id/rooms — active rooms only.
func (h *LocationHandler) ListRooms(c *gin.Context) {
	items, err := h.uc.ListRooms(c.Request.Context(), c.Param("id"), true)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// resolvedRoomResponse is the flat, human-readable delivery path for a room so
// clients can label a saved room_id without walking the tree.
type resolvedRoomResponse struct {
	RoomID       string `json:"RoomID"`
	RoomCode     string `json:"RoomCode"`
	RoomName     string `json:"RoomName"`
	FloorID      string `json:"FloorID"`
	FloorName    string `json:"FloorName"`
	BuildingID   string `json:"BuildingID"`
	BuildingName string `json:"BuildingName"`
}

// ResolveRoom GET /api/v1/locations/rooms/:id — full building/floor/room path.
func (h *LocationHandler) ResolveRoom(c *gin.Context) {
	r, err := h.uc.ResolveRoom(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, resolvedRoomResponse{
		RoomID:       r.Room.ID,
		RoomCode:     r.Room.Code,
		RoomName:     r.Room.Name,
		FloorID:      r.Floor.ID,
		FloorName:    r.Floor.Name,
		BuildingID:   r.Building.ID,
		BuildingName: r.Building.Name,
	})
}
