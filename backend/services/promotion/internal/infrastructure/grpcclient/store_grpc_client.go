package grpcclient

import (
	"context"

	"project/pkg/trace"
	storev1 "project/proto/store/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StoreOwnership holds the ownership fields returned by the store service.
type StoreOwnership struct {
	Found       bool
	VendorID    string
	OwnerUserID string
	Name        string
}

// StoreClient is the interface the promotion service uses to query store data.
// The concrete impl calls StoreService.GetStoreOwnership; the noop fallback
// returns not-found so promotion startup is not gated on store availability.
type StoreClient interface {
	GetStoreOwnership(ctx context.Context, storeID string) (*StoreOwnership, error)
}

type storeGRPCClient struct {
	client storev1.StoreServiceClient
}

// NewStoreClient dials the store gRPC service at addr and returns a StoreClient.
func NewStoreClient(addr string) (StoreClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, err
	}
	return &storeGRPCClient{client: storev1.NewStoreServiceClient(conn)}, nil
}

func (c *storeGRPCClient) GetStoreOwnership(ctx context.Context, storeID string) (*StoreOwnership, error) {
	resp, err := c.client.GetStoreOwnership(ctx, &storev1.GetStoreOwnershipRequest{StoreId: storeID})
	if err != nil {
		return nil, err
	}
	return &StoreOwnership{
		Found:       resp.GetFound(),
		VendorID:    resp.GetVendorId(),
		OwnerUserID: resp.GetOwnerUserId(),
		Name:        resp.GetName(),
	}, nil
}

// NoopStoreClient is used when the store service is unreachable at startup.
// Every call returns not-found so the promotion service degrades gracefully
// (owner routes return 503) rather than panicking.
type NoopStoreClient struct{}

func (n *NoopStoreClient) GetStoreOwnership(_ context.Context, _ string) (*StoreOwnership, error) {
	return &StoreOwnership{Found: false}, nil
}
