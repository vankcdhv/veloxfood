package main

import (
	"project/pkg/app"
	authjwt "project/pkg/auth/jwt"
	authmw "project/pkg/auth/middleware"
	"project/pkg/auth/remotechecker"
	authstore "project/pkg/auth/store"
	"project/pkg/grpcx"
	userv1 "project/proto/user/v1"

	"github.com/gin-gonic/gin"
)

// buildAuth wires JWT verification (public key only — payment does not issue tokens)
// and a remote permission checker backed by the user-service over gRPC.
func buildAuth(deps app.Dependencies) (gin.HandlerFunc, authmw.PermissionChecker, error) {
	cfg := deps.Config

	pubKey, err := authjwt.LoadPublicKey(cfg.JWT.PublicKeyPath, cfg.JWT.PublicKeyPEM)
	if err != nil {
		return nil, nil, err
	}
	jwtSvc := authjwt.NewRS256Service(nil, pubKey, cfg.JWT.AccessTTL, cfg.JWT.RefreshTTL)

	redisAuthStore, err := authstore.NewRedisAuthStore(cfg.Redis)
	if err != nil {
		return nil, nil, err
	}

	conn, err := grpcx.Dial(cfg.UserService.GRPCAddr)
	if err != nil {
		return nil, nil, err
	}

	checker := remotechecker.New(userv1.NewUserServiceClient(conn))
	return authmw.AuthRequired(jwtSvc, redisAuthStore), checker, nil
}
