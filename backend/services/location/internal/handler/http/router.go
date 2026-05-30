package http

import (
	authmw "project/pkg/auth/middleware"
	v1 "project/services/location/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles handlers + middleware for the location service.
type RouterConfig struct {
	LocationHandler      *v1.LocationHandler
	AdminLocationHandler *v1.AdminLocationHandler
	MeLocationHandler    *v1.MeLocationHandler
	AuthMiddleware       gin.HandlerFunc
	PermChecker          authmw.PermissionChecker
}

func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	api := r.Group("/api/v1")

	if cfg.AuthMiddleware == nil || cfg.PermChecker == nil {
		// Auth not configured (dev without JWT key / user-service) — skip routes.
		return
	}

	protected := api.Group("", cfg.AuthMiddleware)
	perm := func(code string) gin.HandlerFunc {
		return authmw.PermissionRequired(cfg.PermChecker, code)
	}

	// ---- Customer: browse tree (any authenticated user) ----
	if cfg.LocationHandler != nil {
		loc := protected.Group("/locations")
		loc.GET("/buildings", cfg.LocationHandler.ListBuildings)
		loc.GET("/buildings/:id/floors", cfg.LocationHandler.ListFloors)
		loc.GET("/floors/:id/rooms", cfg.LocationHandler.ListRooms)
		loc.GET("/rooms/:id", cfg.LocationHandler.ResolveRoom)
	}

	// ---- Customer: saved locations ----
	if cfg.MeLocationHandler != nil {
		me := protected.Group("/me/locations")
		me.GET("", cfg.MeLocationHandler.List)
		me.POST("", cfg.MeLocationHandler.Add)
		me.DELETE("/:id", cfg.MeLocationHandler.Remove)
		me.PATCH("/:id/default", cfg.MeLocationHandler.SetDefault)
	}

	// ---- Admin: manage tree (perm location.manage) ----
	if cfg.AdminLocationHandler != nil {
		h := cfg.AdminLocationHandler
		mgmt := perm("location.manage")

		b := protected.Group("/admin/buildings", mgmt)
		b.GET("", h.ListBuildings)
		b.POST("", h.CreateBuilding)
		b.PATCH("/:id", h.UpdateBuilding)
		b.DELETE("/:id", h.DeleteBuilding)
		b.GET("/:id/floors", h.ListFloors)
		b.POST("/:id/floors", h.CreateFloor)

		f := protected.Group("/admin/floors", mgmt)
		f.PATCH("/:id", h.UpdateFloor)
		f.DELETE("/:id", h.DeleteFloor)
		f.GET("/:id/rooms", h.ListRooms)
		f.POST("/:id/rooms", h.CreateRoom)

		rm := protected.Group("/admin/rooms", mgmt)
		rm.PATCH("/:id", h.UpdateRoom)
		rm.DELETE("/:id", h.DeleteRoom)
	}
}
