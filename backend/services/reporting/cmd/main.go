package main

import (
	"log/slog"

	"project/pkg/app"
	authmw "project/pkg/auth/middleware"
	"project/pkg/middleware"
	reportevent "project/services/reporting/internal/handler/event"
	handlerhttp "project/services/reporting/internal/handler/http"
	v1 "project/services/reporting/internal/handler/http/v1"
	"project/services/reporting/internal/infrastructure/grpcclient"
	"project/services/reporting/internal/infrastructure/persistence"
	"project/services/reporting/internal/usecase"

	"github.com/gin-gonic/gin"
)

func main() {
	a := app.New("reporting-service").WithConfigPath("config/reporting.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		projRepo      := persistence.NewProjectionGormRepository(deps.DB)
		queryRepo     := persistence.NewQueryGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		// ── Usecases ──────────────────────────────────────────────────────────
		analyticsUC := usecase.NewAnalyticsUsecase(queryRepo)

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{}
		if authMW, checker, err := buildAuth(deps); err != nil {
			slog.Error("reporting: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			// store.approve gates admin BI routes; without it any authenticated
			// user could read platform-wide analytics.
			cfg.AdminPermission = authmw.PermissionRequired(checker, "store.approve")
			cfg.AdminHandler = v1.NewAdminAnalyticsHandler(analyticsUC)

			// Store client lets per-store report routes verify the caller owns
			// the store before exposing its data (nil interface degrades safely).
			var storeResolver v1.StoreOwnershipResolver
			if sc, scErr := grpcclient.NewStoreClient(deps.Config.StoreService.GRPCAddr); scErr != nil {
				slog.Error("reporting: store gRPC client setup failed", "err", scErr)
			} else {
				storeResolver = sc
			}
			cfg.StoreHandler = v1.NewStoreReportHandler(analyticsUC, storeResolver, checker)
		}
		handlerhttp.RegisterRoutes(r, cfg)

		// ── Event consumers ───────────────────────────────────────────────────
		orderHandler         := reportevent.NewOrderEventHandler(deps.DB, projRepo, processedRepo)
		paymentHandler       := reportevent.NewPaymentEventHandler(deps.DB, projRepo, processedRepo)
		reviewVendorHandler  := reportevent.NewReviewVendorEventHandler(deps.DB, projRepo, processedRepo)

		startOrderEventConsumer(a, deps, orderHandler)
		startPaymentEventConsumer(a, deps, paymentHandler)
		startPayoutEventConsumer(a, deps, paymentHandler)
		startReviewEventConsumer(a, deps, reviewVendorHandler)
		startVendorEventConsumer(a, deps, reviewVendorHandler)
	})

	// Reporting has no gRPC server — pure consumer + read HTTP API.
	a.Run()
}
