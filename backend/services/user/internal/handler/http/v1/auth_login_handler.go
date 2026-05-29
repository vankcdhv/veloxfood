package v1

import (
	"project/pkg/response"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

// refreshRequest carries the refresh token; optional because browsers send it
// via the httpOnly refresh_token cookie instead of the JSON body.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// logoutRequest accepts both tokens; both optional because browsers send them
// via httpOnly cookies. Non-browser clients may still pass them in the body.
type logoutRequest struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Login authenticates with email/phone + password and returns a JWT pair.
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	pair, err := h.uc.Login(c.Request.Context(), usecase.LoginInput{
		Identifier: req.Identifier,
		Password:   req.Password,
	})
	if err != nil {
		response.HandleError(c, err)
		return
	}

	setAuthCookies(c, pair, h.cookie)
	respondPair(c, pair)
}

// Refresh rotates the JWT pair using a valid refresh token.
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	// Body is optional — bind best-effort so the cookie path still works when
	// the client sends no JSON body at all.
	_ = c.ShouldBindJSON(&req)

	token := req.RefreshToken
	if token == "" {
		token, _ = c.Cookie(cookieRefreshToken)
	}
	if token == "" {
		response.BadRequest(c, "refresh token is required")
		return
	}

	pair, err := h.uc.Refresh(c.Request.Context(), token)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	setAuthCookies(c, pair, h.cookie)
	respondPair(c, pair)
}

// Logout revokes both tokens for the current session.
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req)

	accessToken := req.AccessToken
	if accessToken == "" {
		accessToken, _ = c.Cookie(cookieAccessToken)
	}
	refreshToken := req.RefreshToken
	if refreshToken == "" {
		refreshToken, _ = c.Cookie(cookieRefreshToken)
	}

	if err := h.uc.Logout(c.Request.Context(), accessToken, refreshToken); err != nil {
		response.HandleError(c, err)
		return
	}

	clearAuthCookies(c, h.cookie)
	response.Success(c, gin.H{"message": "logged out"})
}
