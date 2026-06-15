package http

import (
	"github.com/gin-gonic/gin"

	v1 "project/services/payment/internal/handler/http/v1"
)

// RouterConfig carries the wired handlers for payment HTTP routes.
type RouterConfig struct {
	WalletHandler *v1.WalletHandler
	MoMoHandler   *v1.MoMoHandler
	AdminHandler  *v1.AdminPayoutHandler

	// AuthMiddleware validates Bearer tokens. When nil, customer/admin routes are disabled.
	AuthMiddleware gin.HandlerFunc
	// AdminPermission enforces admin-level permission on /admin routes (payouts,
	// settlements). Without it any authenticated user could move settlement money.
	AdminPermission gin.HandlerFunc
}

// RegisterRoutes mounts all payment API routes onto r.
func RegisterRoutes(r *gin.Engine, cfg RouterConfig) {
	api := r.Group("/api/v1")

	// ── Public (no auth) ──────────────────────────────────────────────────────
	// MoMo IPN is called server-to-server by MoMo — must not require JWT.
	if cfg.MoMoHandler != nil {
		pub := api.Group("/payments/momo")
		pub.POST("/ipn", cfg.MoMoHandler.HandleIPN)
		pub.GET("/return", cfg.MoMoHandler.HandleReturn)
	}

	if cfg.AuthMiddleware == nil {
		return
	}

	// ── Authenticated customer routes ─────────────────────────────────────────
	auth := api.Group("/", cfg.AuthMiddleware)
	if cfg.WalletHandler != nil {
		auth.GET("/me/wallet", cfg.WalletHandler.GetMyWallet)
		auth.POST("/wallet/topup", cfg.WalletHandler.InitiateTopup)
	}

	// Store owner earnings overview (reuses payout summary/history).
	if cfg.AdminHandler != nil {
		auth.GET("/me/store-revenue", cfg.AdminHandler.GetStoreRevenue)
	}

	// ── Admin routes ──────────────────────────────────────────────────────────
	if cfg.AdminHandler != nil {
		adminMW := []gin.HandlerFunc{cfg.AuthMiddleware}
		if cfg.AdminPermission != nil {
			adminMW = append(adminMW, cfg.AdminPermission)
		}
		admin := api.Group("/admin", adminMW...)
		admin.GET("/payouts", cfg.AdminHandler.ListPayouts)
		admin.POST("/payouts", cfg.AdminHandler.CreatePayout)
		admin.POST("/payouts/:id/execute", cfg.AdminHandler.ExecutePayout)
		admin.GET("/settlements", cfg.AdminHandler.GetSettlements)
	}
}
