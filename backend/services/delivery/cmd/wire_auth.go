package main

import (
	"project/pkg/app"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	remotechecker "project/pkg/auth/remotechecker"
	authstore "project/pkg/auth/store"
	"project/pkg/trace"
	userv1 "project/proto/user/v1"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// buildAuth wires JWT verification (public key only — delivery does not issue tokens)
// and a remote permission checker backed by the user-service over gRPC.
// Returns: authMiddleware, shipperPermission, adminPermission, error.
func buildAuth(deps app.Dependencies) (gin.HandlerFunc, gin.HandlerFunc, gin.HandlerFunc, error) {
	cfg := deps.Config

	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		return nil, nil, nil, err
	}
	jwtSvc := authjwt.NewRS256Service(nil, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	redisAuthStore, err := authstore.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return nil, nil, nil, err
	}

	conn, err := grpc.NewClient(
		cfg.UserService.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	checker := remotechecker.New(userv1.NewUserServiceClient(conn))
	authMW := authmw.AuthRequired(jwtSvc, redisAuthStore)
	shipperPerm := authmw.PermissionRequired(checker, "SHIPPER")
	adminPerm := authmw.PermissionRequired(checker, "store.approve")

	return authMW, shipperPerm, adminPerm, nil
}
