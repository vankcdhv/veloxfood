package http

import (
	authmw "project/pkg/auth/middleware"
	v1 "project/services/store/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles all handlers and middleware for the store service.
type RouterConfig struct {
	BrowseHandler  *v1.StoreBrowseHandler
	VendorHandler  *v1.VendorStoreHandler
	AdminHandler   *v1.AdminStoreHandler
	AuthMiddleware gin.HandlerFunc
	PermChecker    authmw.PermissionChecker
}

// RegisterRoutes mounts all store service routes onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	api := r.Group("/api/v1")

	if cfg.AuthMiddleware == nil || cfg.PermChecker == nil {
		// Auth not configured — skip all protected routes (dev without JWT).
		return
	}

	protected := api.Group("", cfg.AuthMiddleware)

	perm := func(code string) gin.HandlerFunc {
		return authmw.PermissionRequired(cfg.PermChecker, code)
	}

	// ── Public browse (any authenticated user) ────────────────────────────────
	if cfg.BrowseHandler != nil {
		h := cfg.BrowseHandler
		stores := protected.Group("/stores")
		stores.GET("", h.ListStores)
		// Static paths must be registered BEFORE the wildcard /:id to win routing.
		stores.GET("/mine", h.ListMyStores)
		stores.GET("/:id", h.GetStore)
		stores.GET("/:id/hours", h.GetStoreHours)
		stores.GET("/:id/menu", h.GetMenu)
		stores.GET("/:id/categories", h.ListCategories)
		stores.GET("/:id/ship-fee", h.GetShipFee)
	}

	// ── Vendor owner: manage their own store ──────────────────────────────────
	// Permission is checked per-handler against the store's vendor_id (not a
	// route-level middleware) because we need the store record to know vendor_id.
	if cfg.VendorHandler != nil {
		h := cfg.VendorHandler
		s := protected.Group("/stores/:id")

		// Store settings
		s.PATCH("", h.UpdateStore)
		s.PATCH("/sale-status", h.UpdateSaleStatus)
		s.PATCH("/pickup", h.UpdatePickup)

		// Categories
		s.POST("/categories", h.CreateCategory)
		s.PATCH("/categories/:catId", h.UpdateCategory)
		s.DELETE("/categories/:catId", h.DeleteCategory)

		// Menu items
		s.POST("/menu", h.CreateMenuItem)
		s.PATCH("/menu/:itemId", h.UpdateMenuItem)
		s.PATCH("/menu/:itemId/status", h.ToggleMenuItemStatus)
		s.POST("/menu/:itemId/image", h.UploadMenuItemImage)
		s.DELETE("/menu/:itemId", h.DeleteMenuItem)

		// MenuItem ↔ OptionGroup links
		s.GET("/menu/:itemId/option-groups", h.ListOptionGroupsForItem)
		s.POST("/menu/:itemId/option-groups", h.AttachOptionGroup)
		s.DELETE("/menu/:itemId/option-groups/:ogId", h.DetachOptionGroup)

		// Option groups
		s.GET("/option-groups", h.ListOptionGroups)
		s.POST("/option-groups", h.CreateOptionGroup)
		s.PATCH("/option-groups/:ogId", h.UpdateOptionGroup)
		s.DELETE("/option-groups/:ogId", h.DeleteOptionGroup)

		// Options (children of an option group)
		s.POST("/option-groups/:ogId/options", h.CreateOption)
		s.GET("/option-groups/:ogId/options", h.ListOptions)
		s.PATCH("/option-groups/:ogId/options/:optId", h.UpdateOption)
		s.DELETE("/option-groups/:ogId/options/:optId", h.DeleteOption)

		// Combos
		s.GET("/combos", h.ListCombos)
		s.POST("/combos", h.CreateCombo)
		s.PATCH("/combos/:comboId", h.UpdateCombo)
		s.DELETE("/combos/:comboId", h.DeleteCombo)

		// Combo items
		s.POST("/combos/:comboId/items", h.AddComboItem)
		s.GET("/combos/:comboId/items", h.ListComboItems)
		s.DELETE("/combos/:comboId/items/:menuItemId", h.RemoveComboItem)

		// Ship fee rules
		s.GET("/ship-fees", h.ListShipFeeRules)
		s.POST("/ship-fees", h.CreateShipFeeRule)
		s.PATCH("/ship-fees/:ruleId", h.UpdateShipFeeRule)
		s.DELETE("/ship-fees/:ruleId", h.DeleteShipFeeRule)

		// Hours change request
		s.GET("/hours-change", h.ListHoursChange)
		s.POST("/hours-change", h.SubmitHoursChange)
	}

	// ── Admin: global store.approve permission ────────────────────────────────
	if cfg.AdminHandler != nil {
		h := cfg.AdminHandler
		approve := perm("store.approve")

		adm := protected.Group("/admin/stores", approve)
		adm.GET("", h.ListStores)
		adm.POST("", h.CreateStore)
		adm.GET("/:id/hours-change", h.ListHoursChange)
		adm.PATCH("/:id", h.UpdateStore)
		adm.POST("/:id/hours-change/:req/approve", h.ApproveHoursChange)
		adm.POST("/:id/hours-change/:req/reject", h.RejectHoursChange)
	}
}
