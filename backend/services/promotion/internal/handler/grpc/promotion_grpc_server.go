package grpc

import (
	"context"

	promotionv1 "project/proto/promotion/v1"
	"project/services/promotion/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PromotionServiceServer implements the proto-generated gRPC PromotionService.
type PromotionServiceServer struct {
	promotionv1.UnimplementedPromotionServiceServer
	applyUC usecase.ApplyUsecase
}

func NewPromotionServiceServer(applyUC usecase.ApplyUsecase) *PromotionServiceServer {
	return &PromotionServiceServer{applyUC: applyUC}
}

// ApplyPromotion reserves one or more promotion codes for an order.
// The call is idempotent by order_id.
func (s *PromotionServiceServer) ApplyPromotion(ctx context.Context, req *promotionv1.ApplyPromotionRequest) (*promotionv1.ApplyPromotionResponse, error) {
	result, err := s.applyUC.ApplyPromotion(ctx, usecase.ApplyRequest{
		OrderID:    req.GetOrderId(),
		StoreID:    req.GetStoreId(),
		CustomerID: req.GetCustomerId(),
		Codes:      req.GetCodes(),
		Subtotal:   req.GetSubtotal(),
		ItemCount:  req.GetItemCount(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "apply promotion failed")
	}

	protoApplied := make([]*promotionv1.AppliedCode, len(result.Applied))
	for i, a := range result.Applied {
		protoApplied[i] = &promotionv1.AppliedCode{
			Code:   a.Code,
			Type:   a.Type,
			Amount: a.Amount,
		}
	}

	return &promotionv1.ApplyPromotionResponse{
		Success:      result.Success,
		ItemDiscount: result.ItemDiscount,
		ShipDiscount: result.ShipDiscount,
		Applied:      protoApplied,
		ErrorReason:  result.ErrorReason,
	}, nil
}

// QuotePromotion computes the discount breakdown without reserving quota —
// the order service prices the order before opening a saga.
func (s *PromotionServiceServer) QuotePromotion(ctx context.Context, req *promotionv1.QuotePromotionRequest) (*promotionv1.QuotePromotionResponse, error) {
	result, err := s.applyUC.QuotePromotion(ctx, usecase.ApplyRequest{
		StoreID:    req.GetStoreId(),
		CustomerID: req.GetCustomerId(),
		Codes:      req.GetCodes(),
		Subtotal:   req.GetSubtotal(),
		ItemCount:  req.GetItemCount(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "quote promotion failed")
	}

	protoApplied := make([]*promotionv1.AppliedCode, len(result.Applied))
	for i, a := range result.Applied {
		protoApplied[i] = &promotionv1.AppliedCode{
			Code:   a.Code,
			Type:   a.Type,
			Amount: a.Amount,
		}
	}

	return &promotionv1.QuotePromotionResponse{
		Success:      result.Success,
		ItemDiscount: result.ItemDiscount,
		ShipDiscount: result.ShipDiscount,
		Applied:      protoApplied,
		ErrorReason:  result.ErrorReason,
	}, nil
}

// ConfirmUsage transitions RESERVED usages for an order to CONFIRMED.
func (s *PromotionServiceServer) ConfirmUsage(ctx context.Context, req *promotionv1.ConfirmUsageRequest) (*promotionv1.ConfirmUsageResponse, error) {
	if err := s.applyUC.ConfirmUsage(ctx, req.GetOrderId()); err != nil {
		return nil, status.Error(codes.Internal, "confirm usage failed")
	}
	return &promotionv1.ConfirmUsageResponse{Success: true}, nil
}

// ReleaseUsage voids RESERVED usages and decrements used_count.
func (s *PromotionServiceServer) ReleaseUsage(ctx context.Context, req *promotionv1.ReleaseUsageRequest) (*promotionv1.ReleaseUsageResponse, error) {
	if err := s.applyUC.ReleaseUsage(ctx, req.GetOrderId()); err != nil {
		return nil, status.Error(codes.Internal, "release usage failed")
	}
	return &promotionv1.ReleaseUsageResponse{Success: true}, nil
}
