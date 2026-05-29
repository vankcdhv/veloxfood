package v1

import (
	"net/http"
	"time"

	authjwt "project/pkg/auth/jwt"

	"github.com/gin-gonic/gin"
)

// Cookie names for the httpOnly auth cookies issued by the backend.
const (
	cookieAccessToken  = "access_token"
	cookieRefreshToken = "refresh_token"
)

// CookieConfig controls how auth cookies are emitted.
// Secure must be true under HTTPS; Domain empty means host-only.
type CookieConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Secure     bool
	Domain     string
}

// setAuthCookies writes both auth cookies from a freshly issued token pair.
// Both use Path="/" so the route guard can read refresh_token on any navigation
// and the access cookie is sent to every protected API route. SameSite=Lax
// mitigates CSRF on top-level navigations while keeping same-site XHR working.
func setAuthCookies(c *gin.Context, p authjwt.TokenPair, cfg CookieConfig) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieAccessToken, p.Access, int(p.AccessTTL.Seconds()), "/", cfg.Domain, cfg.Secure, true)
	c.SetCookie(cookieRefreshToken, p.Refresh, int(p.RefreshTTL.Seconds()), "/", cfg.Domain, cfg.Secure, true)
}

// clearAuthCookies expires both auth cookies (used on logout).
func clearAuthCookies(c *gin.Context, cfg CookieConfig) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieAccessToken, "", -1, "/", cfg.Domain, cfg.Secure, true)
	c.SetCookie(cookieRefreshToken, "", -1, "/", cfg.Domain, cfg.Secure, true)
}
