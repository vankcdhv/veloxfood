package v1

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"project/pkg/oauth"
	"project/services/user/internal/usecase"

	"github.com/gin-gonic/gin"
)

const cookieOAuthState = "oauth_state"

// AuthOAuthHandler handles the Google OAuth login + callback flow.
type AuthOAuthHandler struct {
	client  *oauth.Client
	oauthUC usecase.OAuthUsecase
	cookie  CookieConfig
	webURL  string // frontend URL to return to after login
}

func NewAuthOAuthHandler(client *oauth.Client, oauthUC usecase.OAuthUsecase, cookie CookieConfig, webURL string) *AuthOAuthHandler {
	return &AuthOAuthHandler{client: client, oauthUC: oauthUC, cookie: cookie, webURL: webURL}
}

// GoogleLogin GET /auth/google/login — redirect to Google consent screen.
func (h *AuthOAuthHandler) GoogleLogin(c *gin.Context) {
	state, err := randomState()
	if err != nil {
		c.Redirect(http.StatusFound, h.webURL+"/login?error=oauth_state")
		return
	}
	// Short-lived host-only state cookie for CSRF protection on the callback.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieOAuthState, state, 300, "/", h.cookie.Domain, h.cookie.Secure, true)
	c.Redirect(http.StatusFound, h.client.AuthCodeURL(state))
}

// GoogleCallback GET /auth/google/callback — exchange code, issue cookies, return to FE.
func (h *AuthOAuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	saved, _ := c.Cookie(cookieOAuthState)
	c.SetCookie(cookieOAuthState, "", -1, "/", h.cookie.Domain, h.cookie.Secure, true)

	if code == "" || state == "" || saved == "" || state != saved {
		c.Redirect(http.StatusFound, h.webURL+"/login?error=oauth_state")
		return
	}

	gu, err := h.client.Exchange(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusFound, h.webURL+"/login?error=oauth_exchange")
		return
	}

	pair, err := h.oauthUC.GoogleLogin(c.Request.Context(), usecase.GoogleLoginInput{
		Subject:  gu.Sub,
		Email:    gu.Email,
		FullName: gu.Name,
	})
	if err != nil {
		c.Redirect(http.StatusFound, h.webURL+"/login?error=oauth_login")
		return
	}

	setAuthCookies(c, pair, h.cookie)
	c.Redirect(http.StatusFound, h.webURL+"/")
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
