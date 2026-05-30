package grpc

import (
	"context"
	"errors"
	"time"

	"project/pkg/apperror"
	storev1 "project/proto/store/v1"
	"project/services/store/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StoreServiceServer implements the proto-generated gRPC StoreService.
type StoreServiceServer struct {
	storev1.UnimplementedStoreServiceServer
	storeForOrderUC usecase.StoreForOrderUsecase
	quotaUC         usecase.QuotaUsecase
}

func NewStoreServiceServer(
	storeForOrderUC usecase.StoreForOrderUsecase,
	quotaUC usecase.QuotaUsecase,
) *StoreServiceServer {
	return &StoreServiceServer{
		storeForOrderUC: storeForOrderUC,
		quotaUC:         quotaUC,
	}
}

// GetStoreForOrder assembles store availability, ship fee, and menu snapshot.
func (s *StoreServiceServer) GetStoreForOrder(ctx context.Context, req *storev1.GetStoreForOrderRequest) (*storev1.GetStoreForOrderResponse, error) {
	result, err := s.storeForOrderUC.GetStoreForOrder(ctx, req.GetStoreId(), req.GetRoomId())
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return nil, status.Error(codes.InvalidArgument, appErr.Message)
		}
		return nil, status.Error(codes.Internal, "get store for order failed")
	}

	if !result.Found {
		return &storev1.GetStoreForOrderResponse{Found: false}, nil
	}

	protoItems := make([]*storev1.OrderItem, len(result.Items))
	for i, it := range result.Items {
		protoItems[i] = &storev1.OrderItem{
			ItemId: it.ItemID,
			Name:   it.Name,
			Price:  it.Price,
		}
	}

	return &storev1.GetStoreForOrderResponse{
		Found:         true,
		SaleStatus:    result.SaleStatus,
		UnitShipFee:   result.UnitShipFee,
		Served:        result.Served,
		Items:         protoItems,
		OrderDeadline: result.OrderDeadline,
	}, nil
}

// DecrementSlotQuota atomically reduces sold_count for a quota slot.
func (s *StoreServiceServer) DecrementSlotQuota(ctx context.Context, req *storev1.DecrementSlotQuotaRequest) (*storev1.DecrementSlotQuotaResponse, error) {
	date, err := time.Parse("2006-01-02", req.GetDate())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid date format (expected YYYY-MM-DD): %v", err)
	}

	err = s.quotaUC.DecrementSlot(ctx, req.GetItemId(), date, req.GetCutoffId(), int(req.GetQty()))
	if err != nil {
		if err == usecase.ErrQuotaExceeded {
			return &storev1.DecrementSlotQuotaResponse{Ok: false, Remaining: 0}, nil
		}
		return nil, status.Error(codes.Internal, "decrement slot quota failed")
	}

	return &storev1.DecrementSlotQuotaResponse{Ok: true}, nil
}
