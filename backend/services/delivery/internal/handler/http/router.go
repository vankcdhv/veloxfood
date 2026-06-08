package http

import (
	v1 "project/services/delivery/internal/handler/http/v1"

	"github.com/gin-gonic/gin"
)

// RouterConfig bundles handlers and middleware for the delivery service.
type RouterConfig struct {
	// AuthMiddleware validates JWT and injects user_id into context.
	AuthMiddleware gin.HandlerFunc
	// ShipperPermission enforces SHIPPER role for shipper endpoints.
	ShipperPermission gin.HandlerFunc
	// AdminPermission enforces admin-level permission for admin endpoints.
	AdminPermission gin.HandlerFunc

	ShipperHandler *v1.ShipperDeliveryHandler
	AdminHandler   *v1.AdminDeliveryHandler
}

// RegisterRoutes mounts all delivery service routes onto the engine.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	if cfg.AuthMiddleware == nil {
		return
	}

	api := r.Group("/api/v1", cfg.AuthMiddleware)

	// ── Shipper endpoints (require SHIPPER role) ───────────────────────────────
	if cfg.ShipperHandler != nil {
		h := cfg.ShipperHandler

		// available list is readable by any authenticated shipper
		shipperMW := []gin.HandlerFunc{}
		if cfg.ShipperPermission != nil {
			shipperMW = append(shipperMW, cfg.ShipperPermission)
		}

		api.GET("/deliveries/available", append(shipperMW, h.ListAvailable)...)
		api.POST("/deliveries/:orderId/claim", append(shipperMW, h.ClaimDelivery)...)
		api.PATCH("/deliveries/:orderId/status", append(shipperMW, h.UpdateStatus)...)
		api.POST("/deliveries/:orderId/incident-photo", append(shipperMW, h.UploadIncidentPhoto)...)
		api.POST("/deliveries/:orderId/incident", append(shipperMW, h.ReportIncident)...)
		api.GET("/me/deliveries", append(shipperMW, h.MyDeliveries)...)
	}

	// ── Admin endpoints ────────────────────────────────────────────────────────
	if cfg.AdminHandler != nil {
		h := cfg.AdminHandler

		adminMW := []gin.HandlerFunc{}
		if cfg.AdminPermission != nil {
			adminMW = append(adminMW, cfg.AdminPermission)
		}

		api.GET("/admin/deliveries", append(adminMW, h.ListDeliveries)...)
		api.GET("/admin/incidents", append(adminMW, h.ListIncidents)...)
	}
}
