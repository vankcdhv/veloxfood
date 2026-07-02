package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/grpcx"
	userv1 "project/proto/user/v1"
)

// UserClient implements usecase.UserDirectory by calling the user service over gRPC.
type UserClient struct {
	client userv1.UserServiceClient
}

// NewUserClient dials the user service and returns a UserClient.
// addr is typically cfg.UserService.GRPCAddr.
func NewUserClient(addr string) (*UserClient, error) {
	conn, err := grpcx.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("user grpc client: dial %s: %w", addr, err)
	}
	return &UserClient{client: userv1.NewUserServiceClient(conn)}, nil
}

// GetUserNames resolves a batch of user IDs to their full names.
// Returns a map[id]fullName. Unknown IDs are omitted from the map.
func (c *UserClient) GetUserNames(ctx context.Context, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	resp, err := c.client.GetUserBatch(ctx, &userv1.GetUserBatchRequest{Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("user.GetUserBatch: %w", err)
	}
	result := make(map[string]string, len(resp.GetUsers()))
	for _, u := range resp.GetUsers() {
		result[u.GetId()] = u.GetFullName()
	}
	return result, nil
}
