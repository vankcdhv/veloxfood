package http

import (
	v1 "project/services/notification/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles handlers and middleware for the notification service.
type RouterConfig struct {
	AuthMiddleware gin.HandlerFunc
	Handler        *v1.NotificationHandler
}

// RegisterRoutes mounts all notification service routes onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	if cfg.AuthMiddleware == nil || cfg.Handler == nil {
		return
	}

	api := r.Group("/api/v1", cfg.AuthMiddleware)
	h := cfg.Handler

	// ── Notification centre ───────────────────────────────────────────────────
	api.GET("/me/notifications", h.ListNotifications)
	api.GET("/me/notifications/unread-count", h.UnreadCount)
	api.PATCH("/me/notifications/:id/read", h.MarkRead)
	api.POST("/me/notifications/read-all", h.MarkAllRead)

	// ── Device token registration ─────────────────────────────────────────────
	api.POST("/me/device-tokens", h.RegisterDeviceToken)
	api.DELETE("/me/device-tokens/:id", h.DeleteDeviceToken)
}
