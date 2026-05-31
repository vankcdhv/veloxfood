package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/trace"
	storev1 "project/proto/store/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StoreForOrderResult is the response from Store.GetStoreForOrder.
type StoreForOrderResult struct {
	Found         bool
	SaleStatus    string
	UnitShipFee   int64
	Served        bool
	Items         []StoreOrderItem
	OrderDeadline string // ISO-8601 datetime; empty when no cutoff configured
}

// StoreOrderItem is one menu item returned from the store catalog snapshot.
type StoreOrderItem struct {
	ItemID string
	Name   string
	Price  int64
}

// StoreClient wraps the Store gRPC service for use by the Order saga.
type StoreClient struct {
	client storev1.StoreServiceClient
}

// NewStoreClient dials the store service and returns a StoreClient.
func NewStoreClient(addr string) (*StoreClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(trace.UnaryClientInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("store grpc client: dial %s: %w", addr, err)
	}
	return &StoreClient{client: storev1.NewStoreServiceClient(conn)}, nil
}

// GetStoreForOrder fetches store availability, ship fee, and menu snapshot.
func (c *StoreClient) GetStoreForOrder(ctx context.Context, storeID, roomID string) (*StoreForOrderResult, error) {
	resp, err := c.client.GetStoreForOrder(ctx, &storev1.GetStoreForOrderRequest{
		StoreId: storeID,
		RoomId:  roomID,
	})
	if err != nil {
		return nil, fmt.Errorf("store.GetStoreForOrder: %w", err)
	}
	items := make([]StoreOrderItem, len(resp.GetItems()))
	for i, it := range resp.GetItems() {
		items[i] = StoreOrderItem{ItemID: it.GetItemId(), Name: it.GetName(), Price: it.GetPrice()}
	}
	return &StoreForOrderResult{
		Found:         resp.GetFound(),
		SaleStatus:    resp.GetSaleStatus(),
		UnitShipFee:   resp.GetUnitShipFee(),
		Served:        resp.GetServed(),
		Items:         items,
		OrderDeadline: resp.GetOrderDeadline(),
	}, nil
}

// DecrementSlotQuota reduces the sold count for a quota slot. Returns ok=false when
// the slot is already exhausted (not an error — caller should compensate).
func (c *StoreClient) DecrementSlotQuota(ctx context.Context, itemID, cutoffID, date string, qty int) (bool, error) {
	resp, err := c.client.DecrementSlotQuota(ctx, &storev1.DecrementSlotQuotaRequest{
		ItemId:   itemID,
		CutoffId: cutoffID,
		Date:     date,
		Qty:      int32(qty),
	})
	if err != nil {
		return false, fmt.Errorf("store.DecrementSlotQuota: %w", err)
	}
	return resp.GetOk(), nil
}

// RestoreSlotQuota reverses a DecrementSlotQuota (saga compensation / cancel).
func (c *StoreClient) RestoreSlotQuota(ctx context.Context, itemID, cutoffID, date string, qty int) error {
	_, err := c.client.RestoreSlotQuota(ctx, &storev1.RestoreSlotQuotaRequest{
		ItemId:   itemID,
		CutoffId: cutoffID,
		Date:     date,
		Qty:      int32(qty),
	})
	if err != nil {
		return fmt.Errorf("store.RestoreSlotQuota: %w", err)
	}
	return nil
}
