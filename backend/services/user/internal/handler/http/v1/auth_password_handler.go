package v1

import (
	"project/pkg/response"

	"github.com/gin-gonic/gin"
)

type changePasswordRequest struct {
	// TODO Phase 03: extract userID from JWT middleware context instead of body.
	UserID      string `json:"user_id" binding:"required"`
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type forgotPasswordRequest struct {
	Identifier string `json:"identifier" binding:"required"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword validates old password and updates to new.
// POST /api/v1/auth/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.uc.ChangePassword(c.Request.Context(), req.UserID, req.OldPassword, req.NewPassword); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "password changed"})
}

// ForgotPassword triggers a password reset email. Always returns 200 to avoid enumeration.
// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Intentional silent success regardless of whether identifier exists.
	_ = h.uc.ForgotPassword(c.Request.Context(), req.Identifier)
	response.Success(c, gin.H{"message": "if your account exists, a reset link has been sent"})
}

// ResetPassword consumes the one-time reset token and sets the new password.
// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.uc.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.HandleError(c, err)
		return
	}

	response.Success(c, gin.H{"message": "password reset successful"})
}
