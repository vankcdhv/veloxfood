package v1

import (
	"strconv"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminShipperHandler handles admin approval of shipper applications.
type AdminShipperHandler struct {
	adminShipperUC usecase.AdminShipperUsecase
}

func NewAdminShipperHandler(uc usecase.AdminShipperUsecase) *AdminShipperHandler {
	return &AdminShipperHandler{adminShipperUC: uc}
}

// List GET /api/v1/admin/shippers?status=pending&page=1&page_size=20
func (h *AdminShipperHandler) List(c *gin.Context) {
	var status *entity.ShipperStatus
	if s := c.Query("status"); s != "" {
		st := entity.ShipperStatus(s)
		status = &st
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	items, total, err := h.adminShipperUC.ListPendingEnriched(c.Request.Context(), status, page, pageSize)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

// Approve POST /api/v1/admin/shippers/:user_id/approve
func (h *AdminShipperHandler) Approve(c *gin.Context) {
	adminID := authmw.UserIDFromContext(c.Request.Context())
	userID := c.Param("user_id")
	if err := h.adminShipperUC.Approve(c.Request.Context(), adminID, userID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// Reject POST /api/v1/admin/shippers/:user_id/reject  body: {"reason": "..."}
func (h *AdminShipperHandler) Reject(c *gin.Context) {
	adminID := authmw.UserIDFromContext(c.Request.Context())
	userID := c.Param("user_id")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.adminShipperUC.Reject(c.Request.Context(), adminID, userID, req.Reason); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
