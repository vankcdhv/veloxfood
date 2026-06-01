package http

import (
	authmw "project/pkg/auth/middleware"
	v1 "project/services/review/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles all handlers and middleware for the review service.
type RouterConfig struct {
	CustomerHandler *v1.CustomerReviewHandler
	OwnerHandler    *v1.OwnerReviewHandler
	AdminHandler    *v1.AdminReviewHandler
	AuthMiddleware  gin.HandlerFunc
	PermChecker     authmw.PermissionChecker
}

// RegisterRoutes mounts all review service routes onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	if cfg.AuthMiddleware == nil {
		return // auth not configured — skip all protected routes
	}

	api := r.Group("/api/v1", cfg.AuthMiddleware)

	if cfg.CustomerHandler != nil {
		h := cfg.CustomerHandler
		// POST /api/v1/orders/:id/reviews
		api.POST("/orders/:id/reviews", h.CreateReview)
		// GET /api/v1/stores/:id/reviews
		api.GET("/stores/:id/reviews", h.ListStoreReviews)
		// POST /api/v1/reviews/:id/report
		api.POST("/reviews/:id/report", h.ReportReview)
	}

	if cfg.OwnerHandler != nil {
		h := cfg.OwnerHandler
		// POST /api/v1/reviews/:id/reply
		api.POST("/reviews/:id/reply", h.ReplyToReview)
	}

	if cfg.AdminHandler != nil {
		h := cfg.AdminHandler
		admin := api.Group("/admin")
		// GET  /api/v1/admin/reviews/reported
		admin.GET("/reviews/reported", h.ListReported)
		// PATCH /api/v1/admin/reviews/:id/hide
		admin.PATCH("/reviews/:id/hide", h.HideReview)
		// PATCH /api/v1/admin/reviews/:id/restore
		admin.PATCH("/reviews/:id/restore", h.RestoreReview)
	}
}
