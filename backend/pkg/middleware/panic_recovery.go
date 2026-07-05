package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// PanicRecovery replaces gin.Recovery: it converts a handler panic into a 500
// AND logs the stack through slog, so the trace_id-injecting handler ties the
// crash to the request that caused it. gin.Recovery writes to stderr outside
// the structured log stream, which loses both the stack and the correlation.
func PanicRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(c.Request.Context(), "panic recovered",
					"panic", r,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"status":  http.StatusInternalServerError,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
