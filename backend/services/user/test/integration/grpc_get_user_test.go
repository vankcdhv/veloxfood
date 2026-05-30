package integration

import (
	"context"
	"net"
	"testing"

	"project/pkg/audit"
	"project/pkg/config"
	"project/pkg/database"
	userv1 "project/proto/user/v1"
	grpchandler "project/services/user/internal/handler/grpc"
	"project/services/user/internal/infrastructure/persistence"
	"project/services/user/internal/usecase"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestGRPCGetUser_NotFound verifies gRPC server returns error for unknown user ID.
func TestGRPCGetUser_NotFound(t *testing.T) {
	skipIfNoInfra(t)

	cfg, err := config.Load(testConfigPath())
	if err != nil {
		t.Skipf("config load failed: %v", err)
	}

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		t.Skipf("DB unavailable: %v", err)
	}

	userRepo := persistence.NewUserGormRepository(db)
	membershipRepo := persistence.NewVendorMembershipGormRepository(db)
	_ = audit.NoopLogger{} // audit not needed for gRPC handler

	userUC := usecase.NewUserUsecase(userRepo)
	// rbac=nil: this test only exercises GetUser, not CheckPermission.
	srv := grpchandler.NewUserServiceServer(userUC, userRepo, membershipRepo, nil)

	// Start gRPC server on random port
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := grpc.NewServer()
	userv1.RegisterUserServiceServer(s, srv)
	go func() { _ = s.Serve(lis) }()
	t.Cleanup(s.Stop)

	// Dial
	conn, err := grpc.NewClient(lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client := userv1.NewUserServiceClient(conn)

	// GetUser with unknown ID — should return error (not found).
	_, err = client.GetUser(context.Background(), &userv1.GetUserRequest{Id: "00000000-0000-0000-0000-000000000000"})
	if err == nil {
		t.Error("expected error for unknown user ID, got nil")
	}
	t.Logf("gRPC GetUser (unknown id) returned expected error: %v", err)
}
