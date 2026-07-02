package main

import (
	"log/slog"

	"project/pkg/app"
	"project/pkg/middleware"
	"project/pkg/storage"
	reviewv1 "project/proto/review/v1"
	reviewevent "project/services/review/internal/handler/event"
	grpchandler "project/services/review/internal/handler/grpc"
	handlerhttp "project/services/review/internal/handler/http"
	v1 "project/services/review/internal/handler/http/v1"
	"project/services/review/internal/infrastructure/grpcclient"
	"project/services/review/internal/infrastructure/persistence"
	"project/services/review/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("review-service").WithConfigPath("config/review.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		reviewRepo := persistence.NewReviewGormRepository(deps.DB)
		replyRepo := persistence.NewReviewReplyGormRepository(deps.DB)
		reportRepo := persistence.NewReviewReportGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		// ── Cross-service gRPC clients ─────────────────────────────────────────
		orderClient := buildOrderClient(deps)
		storeClient := buildStoreClient(deps)

		// ── Auth & usecase ────────────────────────────────────────────────────
		authMW, permChecker, err := buildAuth(deps)
		if err != nil {
			slog.Error("review: auth setup failed — protected routes disabled", "err", err)
			return
		}

		reviewUC := usecase.NewReviewUsecase(
			deps.DB,
			reviewRepo, replyRepo, reportRepo,
			outboxRepo, processedRepo,
			orderClient, storeClient,
			permChecker,
		)

		// ── Review photo storage (public bucket; nil disables uploads) ────────
		var photoUploader v1.ReviewPhotoUploader
		if client, err := storage.NewClient(deps.Config.MinIO); err != nil {
			slog.Error("review: minio init failed — photo uploads disabled", "err", err)
		} else {
			photoUploader = client
		}

		// ── HTTP router ───────────────────────────────────────────────────────
		handlerhttp.RegisterRoutes(r, handlerhttp.RouterConfig{
			CustomerHandler: v1.NewCustomerReviewHandler(reviewUC, photoUploader),
			OwnerHandler:    v1.NewOwnerReviewHandler(reviewUC),
			AdminHandler:    v1.NewAdminReviewHandler(reviewUC),
			AuthMiddleware:  authMW,
			PermChecker:     permChecker,
		})

		// ── Event consumers ───────────────────────────────────────────────────
		_ = reviewevent.NewOrderEventHandler() // instantiated for type safety
		startOrderEventConsumer(a, deps)

		// ── Background workers ────────────────────────────────────────────────
		startOutboxDispatcher(a, deps)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		reviewRepo := persistence.NewReviewGormRepository(deps.DB)
		replyRepo := persistence.NewReviewReplyGormRepository(deps.DB)
		reportRepo := persistence.NewReviewReportGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		orderClient := buildOrderClient(deps)
		storeClient := buildStoreClient(deps)

		// permChecker for gRPC scope — build independently so gRPC registrar
		// does not depend on RegisterHTTP run order.
		_, permChecker, err := buildAuth(deps)
		if err != nil {
			slog.Error("review: grpc auth setup failed — gRPC server may not check perms", "err", err)
			return
		}

		reviewUC := usecase.NewReviewUsecase(
			deps.DB,
			reviewRepo, replyRepo, reportRepo,
			outboxRepo, processedRepo,
			orderClient, storeClient,
			permChecker,
		)
		reviewv1.RegisterReviewServiceServer(s, grpchandler.NewReviewServiceServer(reviewUC))
	})

	a.Run()
}

// buildOrderClient dials the order service. Returns a noop client on failure so
// review create will return ErrOrderNotFound rather than panicking.
func buildOrderClient(deps app.Dependencies) grpcclient.OrderClient {
	client, err := grpcclient.NewOrderClient(deps.Config.OrderService.GRPCAddr)
	if err != nil {
		slog.Error("review: order grpc client failed — order validation disabled", "err", err)
		return &grpcclient.NoopOrderClient{}
	}
	return client
}

// buildStoreClient dials the store service. Returns a noop client on failure so
// reply routes return 404 rather than panicking.
func buildStoreClient(deps app.Dependencies) grpcclient.StoreClient {
	client, err := grpcclient.NewStoreClient(deps.Config.StoreService.GRPCAddr)
	if err != nil {
		slog.Error("review: store grpc client failed — owner reply authz disabled", "err", err)
		return &grpcclient.NoopStoreClient{}
	}
	return client
}
