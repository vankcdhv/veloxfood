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

	// ── Admin routes ──────────────────────────────────────────────────────────
	if cfg.AdminHandler != nil {
		admin := api.Group("/admin", cfg.AuthMiddleware)
		admin.GET("/payouts", cfg.AdminHandler.ListPayouts)
		admin.POST("/payouts", cfg.AdminHandler.CreatePayout)
		admin.POST("/payouts/:id/execute", cfg.AdminHandler.ExecutePayout)
	}
}
