package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	authjwt "project/pkg/auth/jwt"
	"project/pkg/auth/store"
	"project/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthRequired validates the Bearer token in Authorization header.
// On success it injects user_id and jti into the request context.
// Uses whitelist model: jti must exist in AuthStore (Redis) to be valid.
func AuthRequired(jwtSvc authjwt.JWTService, authStore store.AuthStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		token, ok := extractToken(c)
		if !ok {
			slog.DebugContext(ctx, "auth_required: missing token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
				Error:   "missing or invalid authorization header",
			})
			return
		}

		claims, err := jwtSvc.Parse(ctx, token)
		if err != nil {
			slog.DebugContext(ctx, "auth_required: token parse failed", "err", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
				Error:   "invalid token",
			})
			return
		}

		// Whitelist check — fail closed on Redis error
		exists, err := authStore.Exists(ctx, claims.ID)
		if err != nil {
			slog.ErrorContext(ctx, "auth_required: auth store error", "err", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
				Error:   "token validation failed",
			})
			return
		}
		if !exists {
			slog.DebugContext(ctx, "auth_required: jti not in whitelist", "jti", claims.ID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Response{
				Status:  http.StatusUnauthorized,
				Message: "unauthorized",
				Error:   "token has been revoked",
			})
			return
		}

		// Inject user_id and jti into context
		newCtx := WithUserID(ctx, claims.UserID)
		newCtx = WithJTI(newCtx, claims.ID)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}

// extractToken pulls the access token from the Authorization header, falling
// back to the httpOnly "access_token" cookie set by the backend for browsers.
func extractToken(c *gin.Context) (string, bool) {
	if token, ok := extractBearerToken(c); ok {
		return token, true
	}
	if token, err := c.Cookie("access_token"); err == nil && token != "" {
		return token, true
	}
	return "", false
}

// extractBearerToken parses "Authorization: Bearer <token>" header.
func extractBearerToken(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
