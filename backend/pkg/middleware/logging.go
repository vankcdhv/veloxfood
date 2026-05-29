package middleware

import (
	"log/slog"
	"time"

	"project/pkg/trace"

	"github.com/gin-gonic/gin"
)

func RequestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := trace.ExtractHTTPHeader(c.Request.Header)
		if id == "" {
			id = trace.New()
		}
		ctx := trace.WithTraceID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(trace.HeaderHTTP, id)

		start := time.Now()

		c.Next()

		status := c.Writer.Status()
		latencyMs := time.Since(start).Milliseconds()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Int64("latency_ms", latencyMs),
			slog.String("client_ip", c.ClientIP()),
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("error", c.Errors.String()))
		}

		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		slog.LogAttrs(ctx, level, "http request", attrs...)
	}
}
