package middleware

import (
	"project/pkg/audit"

	"github.com/gin-gonic/gin"
)

// RequestMetadata is a Gin middleware that injects client IP and User-Agent
// into the request context so downstream usecases can write them to audit logs.
// Must run AFTER RequestLogging (which sets trace_id) and BEFORE auth middleware.
func RequestMetadata() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ua := c.Request.UserAgent()

		ctx := audit.WithRequestMeta(c.Request.Context(), ip, ua)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
