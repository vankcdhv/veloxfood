package grpc

import (
	"context"
	"errors"

	"project/pkg/apperror"
	storev1 "project/proto/store/v1"
	"project/services/store/internal/repository"
	"project/services/store/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// StoreServiceServer implements the proto-generated gRPC StoreService.
type StoreServiceServer struct {
	storev1.UnimplementedStoreServiceServer
	storeForOrderUC usecase.StoreForOrderUsecase
	storeRepo       repository.StoreRepository
}

func NewStoreServiceServer(
	storeForOrderUC usecase.StoreForOrderUsecase,
	storeRepo repository.StoreRepository,
) *StoreServiceServer {
	return &StoreServiceServer{
		storeForOrderUC: storeForOrderUC,
		storeRepo:       storeRepo,
	}
}

// GetStoreForOrder assembles store availability, ship fee, and menu snapshot.
func (s *StoreServiceServer) GetStoreForOrder(ctx context.Context, req *storev1.GetStoreForOrderRequest) (*storev1.GetStoreForOrderResponse, error) {
	result, err := s.storeForOrderUC.GetStoreForOrder(ctx, req.GetStoreId(), req.GetLocationLevel(), req.GetRoomId())
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
		Found:          true,
		SaleStatus:     result.SaleStatus,
		UnitShipFee:    result.UnitShipFee,
		Served:         result.Served,
		Items:          protoItems,
		PrepMinutes:    int32(result.PrepMinutes),
		OpenNow:        result.OpenNow,
		OpenTimeToday:  result.OpenTimeToday,
		CloseTimeToday: result.CloseTimeToday,
	}, nil
}

// GetStoreOwnership returns the vendor_id and owner_user_id for a store.
// Promotion service calls this to authorise store-scoped mutations without
// maintaining a local copy of store data.
func (s *StoreServiceServer) GetStoreOwnership(ctx context.Context, req *storev1.GetStoreOwnershipRequest) (*storev1.GetStoreOwnershipResponse, error) {
	store, err := s.storeRepo.GetByID(ctx, req.GetStoreId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &storev1.GetStoreOwnershipResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "get store ownership failed")
	}
	return &storev1.GetStoreOwnershipResponse{
		Found:       true,
		VendorId:    store.VendorID,
		OwnerUserId: store.OwnerUserID,
		Name:        store.Name,
	}, nil
}
