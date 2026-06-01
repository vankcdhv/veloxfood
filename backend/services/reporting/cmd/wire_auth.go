package main

import (
	"project/pkg/app"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	authstore "project/pkg/auth/store"
	"project/pkg/trace"
	userv1 "project/proto/user/v1"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// buildAuth wires JWT verification (public key only — reporting does not issue tokens)
// and a remote permission checker backed by the user-service over gRPC.
func buildAuth(deps app.Dependencies) (gin.HandlerFunc, error) {
	cfg := deps.Config

	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		return nil, err
	}
	jwtSvc := authjwt.NewRS256Service(nil, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	redisAuthStore, err := authstore.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient(
		cfg.UserService.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, err
	}

	_ = userv1.NewUserServiceClient(conn) // dial only — permission checks use authmw directly
	return authmw.AuthRequired(jwtSvc, redisAuthStore), nil
}
