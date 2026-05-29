package v1

import (
	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AdminVendorHandler handles admin-initiated vendor approval endpoints.
type AdminVendorHandler struct {
	adminVendorUC usecase.AdminVendorUsecase
}

func NewAdminVendorHandler(adminVendorUC usecase.AdminVendorUsecase) *AdminVendorHandler {
	return &AdminVendorHandler{adminVendorUC: adminVendorUC}
}

// ApproveVendor POST /admin/vendors/:vendor_id/approve
func (h *AdminVendorHandler) ApproveVendor(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	vendorID := c.Param("vendor_id")

	if err := h.adminVendorUC.ApproveVendor(ctx, adminID, vendorID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}

// RejectVendor POST /admin/vendors/:vendor_id/reject
func (h *AdminVendorHandler) RejectVendor(c *gin.Context) {
	ctx := c.Request.Context()
	adminID := authmw.UserIDFromContext(ctx)
	vendorID := c.Param("vendor_id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.adminVendorUC.RejectVendor(ctx, adminID, vendorID, req.Reason); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, nil)
}
