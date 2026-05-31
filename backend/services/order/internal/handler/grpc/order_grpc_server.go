package grpc

import (
	"context"
	"errors"

	"project/services/order/internal/repository"
	"project/services/order/internal/usecase"
	orderv1 "project/proto/order/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderServiceServer implements the proto-generated gRPC OrderService.
// Called by Payment, Delivery, and Review services to fetch full order data.
type OrderServiceServer struct {
	orderv1.UnimplementedOrderServiceServer
	lifecycleUC usecase.OrderLifecycleUsecase
}

func NewOrderServiceServer(lifecycleUC usecase.OrderLifecycleUsecase) *OrderServiceServer {
	return &OrderServiceServer{lifecycleUC: lifecycleUC}
}

// GetOrder returns full order details including items.
func (s *OrderServiceServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.lifecycleUC.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &orderv1.GetOrderResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "get order failed")
	}

	protoItems := make([]*orderv1.OrderItemProto, len(order.Items))
	for i, it := range order.Items {
		pi := &orderv1.OrderItemProto{
			MenuItemId:   it.MenuItemID,
			NameSnapshot: it.NameSnapshot,
			PriceSnapshot: it.PriceSnapshot,
			Qty:          int32(it.Qty),
		}
		if it.CutoffID != nil {
			pi.CutoffId = *it.CutoffID
		}
		if it.Date != nil {
			pi.Date = it.Date.Format("2006-01-02")
		}
		protoItems[i] = pi
	}

	locationID := ""
	if order.LocationID != nil {
		locationID = *order.LocationID
	}
	pickupPin := ""
	if order.PickupPin != nil {
		pickupPin = *order.PickupPin
	}

	return &orderv1.GetOrderResponse{
		Found:         true,
		OrderId:       order.ID,
		Code:          order.Code,
		CustomerId:    order.CustomerID,
		StoreId:       order.StoreID,
		LocationId:    locationID,
		Fulfillment:   string(order.Fulfillment),
		Status:        string(order.Status),
		PaymentMethod: string(order.PaymentMethod),
		PaymentStatus: string(order.PaymentStatus),
		ItemsTotal:    order.ItemsTotal,
		ShipFee:       order.ShipFee,
		Discount:      order.Discount,
		GrandTotal:    order.GrandTotal,
		PickupPin:     pickupPin,
		PlacedAt:      order.PlacedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Items:         protoItems,
	}, nil
}
