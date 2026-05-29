package v1

import (
	authmw "project/pkg/auth/middleware"
	"project/pkg/response"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// InvitationHandler handles invitation send and accept endpoints.
type InvitationHandler struct {
	staffUC usecase.VendorStaffUsecase
}

// NewInvitationHandler constructs InvitationHandler.
func NewInvitationHandler(staffUC usecase.VendorStaffUsecase) *InvitationHandler {
	return &InvitationHandler{staffUC: staffUC}
}

type sendInviteRequest struct {
	Email        string              `json:"email"          binding:"required,email"`
	RoleInVendor entity.RoleInVendor `json:"role_in_vendor" binding:"required"`
	VendorName   string              `json:"vendor_name"`
}

// Invite handles POST /api/v1/vendors/:vendor_id/invitations.
func (h *InvitationHandler) Invite(c *gin.Context) {
	vendorID := c.Param("vendor_id")
	ownerID := authmw.UserIDFromContext(c.Request.Context())

	var req sendInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	out, err := h.staffUC.Invite(c.Request.Context(), ownerID, vendorID, usecase.InviteStaffInput{
		Email:        req.Email,
		RoleInVendor: req.RoleInVendor,
		VendorName:   req.VendorName,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Created(c, out)
}

type acceptInviteRequest struct {
	Token string `json:"token" binding:"required"`
}

// Accept handles POST /api/v1/invitations/accept.
func (h *InvitationHandler) Accept(c *gin.Context) {
	userID := authmw.UserIDFromContext(c.Request.Context())

	var req acceptInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	out, err := h.staffUC.Accept(c.Request.Context(), userID, usecase.AcceptInvitationInput{
		Token: req.Token,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}
	response.Success(c, out)
}
