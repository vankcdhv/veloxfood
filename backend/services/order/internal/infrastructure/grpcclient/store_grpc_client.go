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
	Found          bool
	SaleStatus     string
	UnitShipFee    int64
	Served         bool
	Items          []StoreOrderItem
	PrepMinutes    int32  // store's minimum preparation time
	OpenNow        bool   // true if store is currently open (SaleStatus=OPEN and within hours)
	OpenTimeToday  string // "HH:MM", empty if closed all day
	CloseTimeToday string // "HH:MM", empty if closed all day
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

// GetStoreForOrder fetches store availability, ship fee, menu snapshot, and open/prep info.
// locationLevel must be "BUILDING", "FLOOR", or "ROOM" (empty ⇒ ROOM back-compat).
// locationID is stored in RoomId per the proto contract regardless of level.
func (c *StoreClient) GetStoreForOrder(ctx context.Context, storeID, locationID, locationLevel string) (*StoreForOrderResult, error) {
	resp, err := c.client.GetStoreForOrder(ctx, &storev1.GetStoreForOrderRequest{
		StoreId:       storeID,
		RoomId:        locationID,
		LocationLevel: locationLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("store.GetStoreForOrder: %w", err)
	}
	items := make([]StoreOrderItem, len(resp.GetItems()))
	for i, it := range resp.GetItems() {
		items[i] = StoreOrderItem{ItemID: it.GetItemId(), Name: it.GetName(), Price: it.GetPrice()}
	}
	return &StoreForOrderResult{
		Found:          resp.GetFound(),
		SaleStatus:     resp.GetSaleStatus(),
		UnitShipFee:    resp.GetUnitShipFee(),
		Served:         resp.GetServed(),
		Items:          items,
		PrepMinutes:    resp.GetPrepMinutes(),
		OpenNow:        resp.GetOpenNow(),
		OpenTimeToday:  resp.GetOpenTimeToday(),
		CloseTimeToday: resp.GetCloseTimeToday(),
	}, nil
}
