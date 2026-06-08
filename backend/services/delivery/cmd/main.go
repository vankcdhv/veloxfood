package main

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"project/pkg/app"
	"project/pkg/middleware"
	"project/pkg/storage"
	deliveryevent "project/services/delivery/internal/handler/event"
	handlerhttp "project/services/delivery/internal/handler/http"
	v1 "project/services/delivery/internal/handler/http/v1"
	"project/services/delivery/internal/infrastructure/grpcclient"
	"project/services/delivery/internal/infrastructure/persistence"
	"project/services/delivery/internal/usecase"

	"github.com/gin-gonic/gin"
)

func main() {
	a := app.New("delivery-service").WithConfigPath("config/delivery.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		deliveryRepo := persistence.NewDeliveryGormRepository(deps.DB)
		batchRepo := persistence.NewBatchGormRepository(deps.DB)
		incidentRepo := persistence.NewIncidentGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		processedRepo := persistence.NewProcessedEventGormRepository(deps.DB)

		// ── Location gRPC client (graceful noop on failure) ───────────────────
		locationClient := buildLocationClient(deps)

		// ── MinIO uploader for incident photos ────────────────────────────────
		uploader := buildUploader(deps)

		// ── Usecases ──────────────────────────────────────────────────────────
		deliveryUC := usecase.NewDeliveryUsecase(deps.DB, deliveryRepo, batchRepo, outboxRepo, locationClient)
		incidentUC := usecase.NewIncidentUsecase(deps.DB, deliveryRepo, incidentRepo, outboxRepo)

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{}
		authMW, shipperPerm, adminPerm, err := buildAuth(deps)
		if err != nil {
			slog.Error("delivery: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.ShipperPermission = shipperPerm
			cfg.AdminPermission = adminPerm
			cfg.ShipperHandler = v1.NewShipperDeliveryHandler(deliveryUC, incidentUC, uploader)
			cfg.AdminHandler = v1.NewAdminDeliveryHandler(deliveryRepo, incidentUC)
		}
		handlerhttp.RegisterRoutes(r, cfg)

		// ── Event consumers ───────────────────────────────────────────────────
		orderHandler := deliveryevent.NewOrderEventHandler(deps.DB, deliveryRepo, processedRepo)
		startOrderEventConsumer(a, deps, orderHandler)

		// ── Background workers ────────────────────────────────────────────────
		startOutboxDispatcher(a, deps)
	})

	a.Run()
}

// buildLocationClient dials the location service. Returns nil on failure —
// available list degrades to showing raw room UUIDs instead of paths.
func buildLocationClient(deps app.Dependencies) *grpcclient.LocationClient {
	addr := deps.Config.LocationService.GRPCAddr
	if addr == "" {
		slog.Warn("delivery: location_service.grpc_addr not configured — room path resolution disabled")
		return nil
	}
	client, err := grpcclient.NewLocationClient(addr)
	if err != nil {
		slog.Error("delivery: location grpc client failed — room paths will show UUIDs", "err", err)
		return nil
	}
	return client
}

// buildUploader builds the MinIO client. Falls back to noopUploader so incident
// photo upload returns a clear error rather than a panic.
func buildUploader(deps app.Dependencies) usecase.FileUploader {
	client, err := storage.NewClient(deps.Config.MinIO)
	if err != nil {
		slog.Error("delivery: minio client failed — incident photo upload disabled", "err", err)
		return &noopUploader{}
	}
	return client
}

// noopUploader returns an error for every upload when MinIO is unavailable.
type noopUploader struct{}

func (n *noopUploader) Put(_ context.Context, _, _ string, _ io.Reader, _ int64) (string, error) {
	return "", errors.New("delivery: object storage unavailable")
}
