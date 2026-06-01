package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/trace"
	orderv1 "project/proto/order/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// OrderResult is the order data needed by the review service.
type OrderResult struct {
	Found      bool
	OrderID    string
	CustomerID string
	StoreID    string
	Status     string // PLACED | CONFIRMED | … | COMPLETED | CANCELLED
}

// OrderClient is the interface the review service uses to validate order eligibility.
type OrderClient interface {
	GetOrder(ctx context.Context, orderID string) (*OrderResult, error)
}

type orderGRPCClient struct{ client orderv1.OrderServiceClient }

// NewOrderClient dials the order service and returns an OrderClient.
func NewOrderClient(addr string) (OrderClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("order grpc client: dial %s: %w", addr, err)
	}
	return &orderGRPCClient{client: orderv1.NewOrderServiceClient(conn)}, nil
}

func (c *orderGRPCClient) GetOrder(ctx context.Context, orderID string) (*OrderResult, error) {
	resp, err := c.client.GetOrder(ctx, &orderv1.GetOrderRequest{OrderId: orderID})
	if err != nil {
		return nil, fmt.Errorf("order.GetOrder: %w", err)
	}
	return &OrderResult{
		Found:      resp.GetFound(),
		OrderID:    resp.GetOrderId(),
		CustomerID: resp.GetCustomerId(),
		StoreID:    resp.GetStoreId(),
		Status:     resp.GetStatus(),
	}, nil
}

// NoopOrderClient is used when the order service is unreachable at startup.
// All calls return not-found so the review service degrades gracefully.
type NoopOrderClient struct{}

func (n *NoopOrderClient) GetOrder(_ context.Context, _ string) (*OrderResult, error) {
	return &OrderResult{Found: false}, nil
}
