package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/trace"
	promotionv1 "project/proto/promotion/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PromotionApplyResult is the outcome of ApplyPromotion.
type PromotionApplyResult struct {
	Success      bool
	ItemDiscount int64
	ShipDiscount int64
	ErrorReason  string
}

// PromotionClient wraps the Promotion gRPC service for use by the Order saga.
type PromotionClient struct {
	client promotionv1.PromotionServiceClient
}

// NewPromotionClient dials the promotion service.
func NewPromotionClient(addr string) (*PromotionClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("promotion grpc client: dial %s: %w", addr, err)
	}
	return &PromotionClient{client: promotionv1.NewPromotionServiceClient(conn)}, nil
}

// ApplyPromotion reserves promotion codes for an order.
// Idempotent by order_id — safe to call on retry.
func (c *PromotionClient) ApplyPromotion(ctx context.Context, orderID, storeID, customerID string, codes []string, subtotal int64, itemCount int32) (*PromotionApplyResult, error) {
	resp, err := c.client.ApplyPromotion(ctx, &promotionv1.ApplyPromotionRequest{
		OrderId:    orderID,
		StoreId:    storeID,
		CustomerId: customerID,
		Codes:      codes,
		Subtotal:   subtotal,
		ItemCount:  itemCount,
	})
	if err != nil {
		return nil, fmt.Errorf("promotion.ApplyPromotion: %w", err)
	}
	return &PromotionApplyResult{
		Success:      resp.GetSuccess(),
		ItemDiscount: resp.GetItemDiscount(),
		ShipDiscount: resp.GetShipDiscount(),
		ErrorReason:  resp.GetErrorReason(),
	}, nil
}

// ConfirmUsage transitions RESERVED → CONFIRMED after order is persisted.
func (c *PromotionClient) ConfirmUsage(ctx context.Context, orderID string) error {
	resp, err := c.client.ConfirmUsage(ctx, &promotionv1.ConfirmUsageRequest{OrderId: orderID})
	if err != nil {
		return fmt.Errorf("promotion.ConfirmUsage: %w", err)
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("promotion.ConfirmUsage returned success=false")
	}
	return nil
}

// ReleaseUsage voids RESERVED usages (saga compensation).
// Idempotent — no-op when no RESERVED rows exist.
func (c *PromotionClient) ReleaseUsage(ctx context.Context, orderID string) error {
	_, err := c.client.ReleaseUsage(ctx, &promotionv1.ReleaseUsageRequest{OrderId: orderID})
	if err != nil {
		return fmt.Errorf("promotion.ReleaseUsage: %w", err)
	}
	return nil
}
