package http

import (
	authmw "project/pkg/auth/middleware"
	v1 "project/services/promotion/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles all handlers and middleware for the promotion service.
type RouterConfig struct {
	VendorHandler  *v1.VendorPromotionHandler
	PublicHandler  *v1.PublicPromotionHandler
	AuthMiddleware gin.HandlerFunc
	PermChecker    authmw.PermissionChecker
}

// RegisterRoutes mounts all promotion service routes onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	api := r.Group("/api/v1")

	if cfg.AuthMiddleware == nil {
		// Auth not configured — skip all protected routes (dev without JWT).
		return
	}

	protected := api.Group("", cfg.AuthMiddleware)

	// ── Vendor owner: manage promotions for their own store ───────────────────
	// Permission is checked per-handler via GetStoreOwnership + HasVendorPermission
	// so vendor_id is verified against the Store service (not a local table).
	if cfg.VendorHandler != nil {
		h := cfg.VendorHandler
		stores := protected.Group("/stores/:storeId/promotions")
		stores.GET("", h.ListPromotions)
		stores.POST("", h.CreatePromotion)
		stores.PATCH("/:promoId", h.UpdatePromotion)
		stores.DELETE("/:promoId", h.DeletePromotion)
	}

	// ── Customer: validate/preview a promotion code (dry-run, no reservation) ─
	if cfg.PublicHandler != nil {
		h := cfg.PublicHandler
		protected.POST("/promotions/validate", h.ValidatePromotion)
	}
}
