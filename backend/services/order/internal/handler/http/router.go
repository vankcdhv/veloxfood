package http

import (
	v1 "project/services/order/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles all handlers and middleware for the order service.
type RouterConfig struct {
	AuthMiddleware   gin.HandlerFunc
	CartHandler      *v1.CustomerCartHandler
	CustomerHandler  *v1.CustomerOrderHandler
	OwnerHandler     *v1.OwnerOrderHandler
}

// RegisterRoutes mounts all order service routes onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	if cfg.AuthMiddleware == nil {
		// Auth not configured — all protected routes are disabled.
		return
	}

	api := r.Group("/api/v1", cfg.AuthMiddleware)

	// ── Customer: cart ────────────────────────────────────────────────────────
	if cfg.CartHandler != nil {
		h := cfg.CartHandler
		api.GET("/me/cart", h.GetCart)
		api.PUT("/me/cart/items", h.AddOrUpdateItem)
		api.DELETE("/me/cart/items/:menuItemId", h.RemoveItem)
		api.DELETE("/me/cart", h.ClearCart)
	}

	// ── Customer: orders ──────────────────────────────────────────────────────
	if cfg.CustomerHandler != nil {
		h := cfg.CustomerHandler
		api.POST("/orders", h.PlaceOrder)
		api.GET("/orders", h.ListMyOrders)
		api.GET("/orders/:id", h.GetMyOrder)
		api.POST("/orders/:id/cancel", h.CancelOrder)
		api.POST("/orders/:id/reorder", h.Reorder)
	}

	// ── Owner / Staff: store order management ─────────────────────────────────
	if cfg.OwnerHandler != nil {
		h := cfg.OwnerHandler
		s := api.Group("/stores/:storeId")
		s.GET("/orders", h.ListStoreOrders)
		s.GET("/orders/:id", h.GetOrder)
		s.POST("/orders/:id/confirm", h.ConfirmOrder)
		s.POST("/orders/:id/reject", h.RejectOrder)
		s.PATCH("/orders/:id/status", h.AdvanceStatus)
		s.POST("/orders/:id/pickup-verify", h.VerifyPickupPIN)
	}
}
