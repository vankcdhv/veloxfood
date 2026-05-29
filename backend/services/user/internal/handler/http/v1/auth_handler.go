package v1

import (
	"net/http"
	"time"

	authjwt "project/pkg/auth/jwt"
	"project/pkg/response"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

// AuthHandler holds the auth usecase and provides helper methods.
type AuthHandler struct {
	uc     usecase.AuthUsecase
	cookie CookieConfig
}

func NewAuthHandler(uc usecase.AuthUsecase, cookie CookieConfig) *AuthHandler {
	return &AuthHandler{uc: uc, cookie: cookie}
}

// tokenPairResponse is the common auth success envelope.
type tokenPairResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"` // seconds until access token expires
	IssuedAt     time.Time `json:"issued_at"`
}

func toTokenPairResponse(p authjwt.TokenPair) tokenPairResponse {
	return tokenPairResponse{
		AccessToken:  p.Access,
		RefreshToken: p.Refresh,
		ExpiresIn:    int64(p.AccessTTL.Seconds()),
		IssuedAt:     time.Now(),
	}
}

// respondPair writes a 200 with the token pair envelope.
func respondPair(c *gin.Context, p authjwt.TokenPair) {
	response.Success(c, toTokenPairResponse(p))
}

// respondPairCreated writes a 201 with the token pair envelope.
func respondPairCreated(c *gin.Context, p authjwt.TokenPair) {
	c.JSON(http.StatusCreated, struct {
		Status int               `json:"status"`
		Data   tokenPairResponse `json:"data"`
	}{http.StatusCreated, toTokenPairResponse(p)})
}
