package grpcclient

import (
	"context"
	"fmt"
	"log/slog"

	"project/pkg/trace"
	userv1 "project/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UserInfo is the minimal display info resolved from the user service.
type UserInfo struct {
	FullName string
	Phone    string
}

// UserClient wraps the User gRPC service so the admin incident view can show a
// shipper's name instead of a raw UUID.
type UserClient struct {
	client userv1.UserServiceClient
}

// NewUserClient dials the user service. Returns nil on failure — callers must
// nil-check and degrade gracefully (the name stays empty).
func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("user grpc client: dial %s: %w", addr, err)
	}
	return &UserClient{client: userv1.NewUserServiceClient(conn)}, nil
}

// GetUser resolves a user's name + phone. Returns a zero UserInfo on any error
// (including a nil client), so callers never need to guard the result.
func (c *UserClient) GetUser(ctx context.Context, userID string) UserInfo {
	if c == nil || userID == "" {
		return UserInfo{}
	}
	resp, err := c.client.GetUser(ctx, &userv1.GetUserRequest{Id: userID})
	if err != nil {
		slog.WarnContext(ctx, "delivery: user.GetUser failed", "user_id", userID, "err", err)
		return UserInfo{}
	}
	u := resp.GetUser()
	if u == nil {
		return UserInfo{}
	}
	return UserInfo{FullName: u.GetFullName(), Phone: u.GetPhone()}
}
