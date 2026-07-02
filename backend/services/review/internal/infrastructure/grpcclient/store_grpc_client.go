package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/grpcx"
	storev1 "project/proto/store/v1"
)

// StoreOwnership holds ownership data needed for reply authorisation.
type StoreOwnership struct {
	Found       bool
	VendorID    string
	OwnerUserID string
}

// StoreClient is the interface the review service uses for store ownership checks.
type StoreClient interface {
	GetStoreOwnership(ctx context.Context, storeID string) (*StoreOwnership, error)
}

type storeGRPCClient struct{ client storev1.StoreServiceClient }

// NewStoreClient dials the store gRPC service at addr and returns a StoreClient.
func NewStoreClient(addr string) (StoreClient, error) {
	conn, err := grpcx.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("store grpc client: dial %s: %w", addr, err)
	}
	return &storeGRPCClient{client: storev1.NewStoreServiceClient(conn)}, nil
}

func (c *storeGRPCClient) GetStoreOwnership(ctx context.Context, storeID string) (*StoreOwnership, error) {
	resp, err := c.client.GetStoreOwnership(ctx, &storev1.GetStoreOwnershipRequest{StoreId: storeID})
	if err != nil {
		return nil, fmt.Errorf("store.GetStoreOwnership: %w", err)
	}
	return &StoreOwnership{
		Found:       resp.GetFound(),
		VendorID:    resp.GetVendorId(),
		OwnerUserID: resp.GetOwnerUserId(),
	}, nil
}

// NoopStoreClient degrades gracefully when store-service is unavailable at startup.
type NoopStoreClient struct{}

func (n *NoopStoreClient) GetStoreOwnership(_ context.Context, _ string) (*StoreOwnership, error) {
	return &StoreOwnership{Found: false}, nil
}
