package http

import (
	"github.com/gin-gonic/gin"

	v1 "project/services/reporting/internal/handler/http/v1"
)

// RouterConfig carries the wired handlers for reporting HTTP routes.
type RouterConfig struct {
	AdminHandler *v1.AdminAnalyticsHandler
	StoreHandler *v1.StoreReportHandler

	// AuthMiddleware validates Bearer tokens. When nil, all protected routes are disabled.
	AuthMiddleware gin.HandlerFunc
	// AdminPermission enforces admin-level permission on /admin analytics routes.
	// Without it any authenticated user could read platform-wide BI data.
	AdminPermission gin.HandlerFunc
}

// RegisterRoutes mounts all reporting API routes onto r.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	if cfg.AuthMiddleware == nil {
		return
	}

	api := r.Group("/api/v1", cfg.AuthMiddleware)

	// ── Admin analytics ───────────────────────────────────────────────────────
	if cfg.AdminHandler != nil {
		admin := api.Group("/admin")
		if cfg.AdminPermission != nil {
			admin.Use(cfg.AdminPermission)
		}
		admin.GET("/analytics", cfg.AdminHandler.GetAnalytics)
		admin.GET("/orders/recent", cfg.AdminHandler.ListRecentOrders)
	}

	// ── Store owner reports ───────────────────────────────────────────────────
	if cfg.StoreHandler != nil {
		stores := api.Group("/stores")
		stores.GET("/:id/reports", cfg.StoreHandler.GetStoreReport)
		stores.GET("/:id/revenue", cfg.StoreHandler.GetStoreRevenue)
	}
}
