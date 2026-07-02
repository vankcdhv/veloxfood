package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/grpcx"
	storev1 "project/proto/store/v1"
)

// StoreOwnership holds the vendor/owner fields used to authorize store-owner
// access to a store's settlement/revenue data.
type StoreOwnership struct {
	Found       bool
	VendorID    string
	OwnerUserID string
	Name        string
}

// StoreClient calls the store service over gRPC.
type StoreClient struct {
	client storev1.StoreServiceClient
}

// NewStoreClient dials the store service and returns a StoreClient.
func NewStoreClient(addr string) (*StoreClient, error) {
	conn, err := grpcx.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("store grpc client: dial %s: %w", addr, err)
	}
	return &StoreClient{client: storev1.NewStoreServiceClient(conn)}, nil
}

// GetStoreOwnership resolves a store's vendor_id so the caller can verify the
// requester owns (manages) the store before exposing its revenue.
func (c *StoreClient) GetStoreOwnership(ctx context.Context, storeID string) (*StoreOwnership, error) {
	resp, err := c.client.GetStoreOwnership(ctx, &storev1.GetStoreOwnershipRequest{StoreId: storeID})
	if err != nil {
		return nil, fmt.Errorf("store.GetStoreOwnership: %w", err)
	}
	return &StoreOwnership{
		Found:       resp.GetFound(),
		VendorID:    resp.GetVendorId(),
		OwnerUserID: resp.GetOwnerUserId(),
		Name:        resp.GetName(),
	}, nil
}
