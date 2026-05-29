package v1

import (
	"log/slog"

	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// VendorHandler handles vendor onboarding and staff management.
type VendorHandler struct {
	onboardUC usecase.VendorOnboardUsecase
	staffUC   usecase.VendorStaffUsecase
}

// NewVendorHandler constructs VendorHandler.
func NewVendorHandler(onboardUC usecase.VendorOnboardUsecase, staffUC usecase.VendorStaffUsecase) *VendorHandler {
	return &VendorHandler{onboardUC: onboardUC, staffUC: staffUC}
}

type onboardRequest struct {
	VendorName   string `json:"vendor_name"   binding:"required"`
	BusinessType string `json:"business_type"`
	Address      string `json:"address"`
	Phone        string `json:"phone"`
}

// Onboard handles POST /api/v1/vendors/onboard.
func (h *VendorHandler) Onboard(c *gin.Context) {
	var req onboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := authmw.UserIDFromContext(c.Request.Context())
	out, err := h.onboardUC.Onboard(c.Request.Context(), userID, usecase.VendorOnboardInput{
		VendorName:   req.VendorName,
		BusinessType: req.BusinessType,
		Address:      req.Address,
		Phone:        req.Phone,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, out)
}

// ListStaff handles GET /api/v1/vendors/:vendor_id/staff.
func (h *VendorHandler) ListStaff(c *gin.Context) {
	vendorID := c.Param("vendor_id")
	var status *entity.VendorMembershipStatus
	if s := c.Query("status"); s != "" {
		v := entity.VendorMembershipStatus(s)
		status = &v
	}

	members, err := h.staffUC.ListStaff(c.Request.Context(), vendorID, status)
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, members)
}

// RemoveStaff handles DELETE /api/v1/vendors/:vendor_id/staff/:user_id.
func (h *VendorHandler) RemoveStaff(c *gin.Context) {
	vendorID := c.Param("vendor_id")
	targetUserID := c.Param("user_id")
	removerID := authmw.UserIDFromContext(c.Request.Context())

	if err := h.staffUC.RemoveStaff(c.Request.Context(), removerID, vendorID, targetUserID); err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, gin.H{"message": "staff member removed"})
}

// MyVendors handles GET /api/v1/me/vendors.
func (h *VendorHandler) MyVendors(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())
	memberships, err := h.staffUC.ListMyVendors(c.Request.Context(), userID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "my vendors: load failed", "user_id", userID, "err", err)
		response.HandleError(c, err)
		return
	}
	response.Success(c, memberships)
}
