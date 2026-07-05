package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/audit"
	"project/pkg/middleware"
	"project/pkg/saga"
	promotionv1 "project/proto/promotion/v1"
	grpchandler "project/services/promotion/internal/handler/grpc"
	handlerhttp "project/services/promotion/internal/handler/http"
	v1 "project/services/promotion/internal/handler/http/v1"
	"project/services/promotion/internal/infrastructure/grpcclient"
	"project/services/promotion/internal/infrastructure/persistence"
	"project/services/promotion/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("promotion-service").WithConfigPath("config/promotion.yaml")

	// Promotion is a DTM saga participant — point the barrier at Postgres.
	saga.Setup()

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		promoRepo := persistence.NewPromotionGormRepository(deps.DB)
		usageRepo := persistence.NewPromotionUsageGormRepository(deps.DB)

		// ── Store gRPC client (ownership checks) ──────────────────────────────
		storeClient := buildStoreClient(deps)

		// ── Usecases ──────────────────────────────────────────────────────────
		promoUC := usecase.NewPromotionUsecase(deps.DB, promoRepo, usageRepo, storeClient, audit.NewGormLogger(deps.DB))

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{
			PublicHandler: v1.NewPublicPromotionHandler(promoUC),
		}
		if authMW, permChecker, err := buildAuth(deps); err != nil {
			slog.Error("promotion: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.PermChecker = permChecker
			cfg.VendorHandler = v1.NewVendorPromotionHandler(promoUC, permChecker)
		}

		handlerhttp.RegisterRoutes(r, cfg)

		// ── Event consumers ───────────────────────────────────────────────────
		// applyUC must be built here (same scope as HTTP wiring) so the consumer
		// has access to it; build its deps independently to avoid RegisterHTTP
		// ordering requirements on RegisterGRPC.
		applyUCForConsumer := usecase.NewApplyUsecase(deps.DB, promoRepo, usageRepo)
		startOrderEventConsumer(a, deps, applyUCForConsumer)

		// ── Background workers ────────────────────────────────────────────────
		startReservationJanitor(a, applyUCForConsumer, deps.Config.Promotion.ReserveTTL)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		// Build repositories independently so gRPC registrar does not depend on
		// RegisterHTTP run order (repositories are stateless value objects).
		promoRepo := persistence.NewPromotionGormRepository(deps.DB)
		usageRepo := persistence.NewPromotionUsageGormRepository(deps.DB)
		applyUC := usecase.NewApplyUsecase(deps.DB, promoRepo, usageRepo)

		promotionv1.RegisterPromotionServiceServer(s, grpchandler.NewPromotionServiceServer(deps.DB, applyUC))
	})

	a.Run()
}

// buildStoreClient dials the store service for ownership resolution. Falls back
// to NoopStoreClient so the promotion service starts even when store-service is
// temporarily unavailable (owner routes will return 404 on unknown stores).
func buildStoreClient(deps app.Dependencies) grpcclient.StoreClient {
	client, err := grpcclient.NewStoreClient(deps.Config.StoreService.GRPCAddr)
	if err != nil {
		slog.Error("promotion: store grpc client failed — store ownership checks disabled", "err", err)
		return &grpcclient.NoopStoreClient{}
	}
	return client
}
