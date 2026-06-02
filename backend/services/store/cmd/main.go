package main

import (
	"context"
	"io"
	"log/slog"

	"project/pkg/app"
	"project/pkg/middleware"
	"project/pkg/storage"
	storev1 "project/proto/store/v1"
	grpchandler "project/services/store/internal/handler/grpc"
	handlerhttp "project/services/store/internal/handler/http"
	v1 "project/services/store/internal/handler/http/v1"
	"project/services/store/internal/infrastructure/grpcclient"
	"project/services/store/internal/infrastructure/persistence"
	"project/services/store/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("store-service").WithConfigPath("config/store.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		// ── Repositories ──────────────────────────────────────────────────────
		storeRepo := persistence.NewStoreGormRepository(deps.DB)
		catalogRepo := persistence.NewCatalogGormRepository(deps.DB)
		shippingRepo := persistence.NewShippingGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)

		// ── Location gRPC client (LocationResolver) ──────────────────────────
		roomResolver := buildLocationResolver(deps)

		// ── User directory (owner name enrichment) ────────────────────────────
		userDir := buildUserDirectory(deps)

		// ── Usecases ──────────────────────────────────────────────────────────
		storeUC := usecase.NewStoreUsecase(deps.DB, storeRepo, outboxRepo, userDir)
		catalogUC := usecase.NewCatalogUsecase(catalogRepo)
		shipFeeUC := usecase.NewShipFeeUsecase(shippingRepo, roomResolver)
		hoursUC := usecase.NewHoursUsecase(deps.DB, shippingRepo, outboxRepo)

		// ── MinIO uploader ────────────────────────────────────────────────────
		uploader := buildUploader(deps)

		// ── HTTP router ───────────────────────────────────────────────────────
		cfg := handlerhttp.RouterConfig{
			BrowseHandler: v1.NewStoreBrowseHandler(storeUC, catalogUC, shipFeeUC, hoursUC),
		}
		if authMW, permChecker, err := buildAuth(deps); err != nil {
			slog.Error("store: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.PermChecker = permChecker
			cfg.VendorHandler = v1.NewVendorStoreHandler(storeUC, catalogUC, shipFeeUC, hoursUC, permChecker, uploader)
			cfg.AdminHandler = v1.NewAdminStoreHandler(storeUC, hoursUC)
		}

		handlerhttp.RegisterRoutes(r, cfg)

		// ── Background workers ────────────────────────────────────────────────
		startOutboxDispatcher(a, deps)
		startVendorEventConsumer(a, deps, storeUC)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		// Build the gRPC-facing usecases independently from deps so this registrar
		// does not depend on RegisterHTTP run order (repositories are stateless).
		storeRepo := persistence.NewStoreGormRepository(deps.DB)
		catalogRepo := persistence.NewCatalogGormRepository(deps.DB)
		shippingRepo := persistence.NewShippingGormRepository(deps.DB)
		outboxRepo := persistence.NewOutboxGormRepository(deps.DB)
		roomResolver := buildLocationResolver(deps)
		hoursUC := usecase.NewHoursUsecase(deps.DB, shippingRepo, outboxRepo)

		storeForOrderUC := usecase.NewStoreForOrderUsecase(storeRepo, catalogRepo, shippingRepo, roomResolver, hoursUC)

		storev1.RegisterStoreServiceServer(s, grpchandler.NewStoreServiceServer(storeForOrderUC, storeRepo))
	})

	a.Run()
}

// buildUserDirectory dials the user service for owner name resolution.
// Falls back to nil (usecase applies a noop internally) if dial fails so the
// store service starts even when user-service is temporarily unavailable.
func buildUserDirectory(deps app.Dependencies) usecase.UserDirectory {
	client, err := grpcclient.NewUserClient(deps.Config.UserService.GRPCAddr)
	if err != nil {
		slog.Error("store: user grpc client failed — OwnerUserName will be empty", "err", err)
		return nil
	}
	return client
}

// buildLocationResolver dials the location service. Falls back to noopLocationResolver
// so the service starts even when location-service is temporarily unavailable.
func buildLocationResolver(deps app.Dependencies) usecase.LocationResolver {
	client, err := grpcclient.NewLocationClient(deps.Config.LocationService.GRPCAddr)
	if err != nil {
		slog.Error("store: location grpc client failed — ship fee resolution disabled", "err", err)
		return &noopLocationResolver{}
	}
	return client
}

// buildUploader builds the MinIO client. Falls back to noopUploader so image
// upload returns a clear error rather than a panic.
func buildUploader(deps app.Dependencies) usecase.FileUploader {
	client, err := storage.NewClient(deps.Config.MinIO)
	if err != nil {
		slog.Error("store: minio client failed — image upload disabled", "err", err)
		return &noopUploader{}
	}
	return client
}

// noopLocationResolver is a stand-in used when the location service is unreachable
// at startup. All fee-resolution calls will return ErrLocationNotServed.
type noopLocationResolver struct{}

func (n *noopLocationResolver) GetRoom(_ context.Context, _ string) (floorID, buildingID string, found bool, err error) {
	return "", "", false, nil
}

func (n *noopLocationResolver) GetFloor(_ context.Context, _ string) (buildingID string, found bool, err error) {
	return "", false, nil
}

func (n *noopLocationResolver) GetBuilding(_ context.Context, _ string) (found bool, err error) {
	return false, nil
}

// noopUploader returns an internal error for every upload when MinIO is
// unavailable. Operators must fix MinIO configuration for image uploads to work.
type noopUploader struct{}

func (n *noopUploader) Put(_ context.Context, _, _ string, _ io.Reader, _ int64) (string, error) {
	return "", usecase.ErrForbidden
}
