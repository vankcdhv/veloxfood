package v1

import (
	"project/pkg/response"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

type registerRequest struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
}

type verifyRegisterRequest struct {
	Destination string `json:"destination" binding:"required"`
	Code        string `json:"code" binding:"required,len=6"`
}

// Register creates a new pending user and sends an OTP.
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Email == "" && req.Phone == "" {
		response.BadRequest(c, "email or phone is required")
		return
	}

	user, err := h.uc.Register(c.Request.Context(), usecase.RegisterInput{
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.Created(c, gin.H{"id": user.ID, "status": user.Status})
}

// VerifyRegister validates the OTP and activates the user account.
// POST /api/v1/auth/verify-register
func (h *AuthHandler) VerifyRegister(c *gin.Context) {
	var req verifyRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	pair, err := h.uc.VerifyRegister(c.Request.Context(), usecase.VerifyRegisterInput{
		Destination: req.Destination,
		Code:        req.Code,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	setAuthCookies(c, pair, h.cookie)
	respondPairCreated(c, pair)
}
