package main

import (
	"log/slog"

	"project/pkg/app"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	"project/pkg/auth/remotechecker"
	"project/pkg/auth/store"
	"project/pkg/grpcx"
	"project/pkg/middleware"
	locationv1 "project/proto/location/v1"
	userv1 "project/proto/user/v1"
	grpchandler "project/services/location/internal/handler/grpc"
	handlerhttp "project/services/location/internal/handler/http"
	v1 "project/services/location/internal/handler/http/v1"
	"project/services/location/internal/infrastructure/persistence"
	"project/services/location/internal/usecase"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func main() {
	a := app.New("location-service").WithConfigPath("config/location.yaml")

	a.RegisterHTTP(func(r *gin.Engine, deps app.Dependencies) {
		r.Use(middleware.RequestMetadata())

		locRepo := persistence.NewLocationGormRepository(deps.DB)
		custRepo := persistence.NewCustomerLocationGormRepository(deps.DB)
		locUC := usecase.NewLocationUsecase(locRepo)
		custUC := usecase.NewCustomerLocationUsecase(custRepo, locRepo)

		cfg := handlerhttp.RouterConfig{
			LocationHandler:      v1.NewLocationHandler(locUC),
			AdminLocationHandler: v1.NewAdminLocationHandler(locUC),
			MeLocationHandler:    v1.NewMeLocationHandler(custUC),
		}

		// Auth (verify-only) + cross-service permission checker.
		if authMW, permChecker, err := buildAuth(deps); err != nil {
			slog.Error("location: auth setup failed — protected routes disabled", "err", err)
		} else {
			cfg.AuthMiddleware = authMW
			cfg.PermChecker = permChecker
		}

		handlerhttp.RegisterRoutes(r, cfg)
	})

	a.RegisterGRPC(func(s *grpc.Server, deps app.Dependencies) {
		locUC := usecase.NewLocationUsecase(persistence.NewLocationGormRepository(deps.DB))
		locationv1.RegisterLocationServiceServer(s, grpchandler.NewLocationServiceServer(locUC))
	})

	a.Run()
}

// buildAuth wires JWT verification (public key) + a remote permission checker
// that delegates to the user-service over gRPC.
func buildAuth(deps app.Dependencies) (gin.HandlerFunc, authmw.PermissionChecker, error) {
	cfg := deps.Config

	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		return nil, nil, err
	}
	jwtSvc := authjwt.NewRS256Service(nil, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	authStore, err := store.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpcx.Dial(cfg.UserService.GRPCAddr)
	if err != nil {
		return nil, nil, err
	}
	checker := remotechecker.New(userv1.NewUserServiceClient(conn))

	return authmw.AuthRequired(jwtSvc, authStore), checker, nil
}
