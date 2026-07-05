package grpc

import (
	"context"

	"project/pkg/saga"
	promotionv1 "project/proto/promotion/v1"
	"project/services/promotion/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// PromotionServiceServer implements the proto-generated gRPC PromotionService.
// Apply/Confirm/Release double as DTM saga branches: when the call carries DTM
// branch metadata it runs under the sub-transaction barrier and signals
// business failure via codes.Aborted (DTM rolls the saga back); direct calls
// (inline engine, consumers) keep the body-level result contract.
type PromotionServiceServer struct {
	promotionv1.UnimplementedPromotionServiceServer
	db      *gorm.DB
	applyUC usecase.ApplyUsecase
}

func NewPromotionServiceServer(db *gorm.DB, applyUC usecase.ApplyUsecase) *PromotionServiceServer {
	return &PromotionServiceServer{db: db, applyUC: applyUC}
}

// ApplyPromotion reserves one or more promotion codes for an order.
// The call is idempotent by order_id.
func (s *PromotionServiceServer) ApplyPromotion(ctx context.Context, req *promotionv1.ApplyPromotionRequest) (*promotionv1.ApplyPromotionResponse, error) {
	ureq := usecase.ApplyRequest{
		OrderID:          req.GetOrderId(),
		StoreID:          req.GetStoreId(),
		CustomerID:       req.GetCustomerId(),
		Codes:            req.GetCodes(),
		Subtotal:         req.GetSubtotal(),
		ItemCount:        req.GetItemCount(),
		ExpectedDiscount: req.GetExpectedDiscount(),
		CheckExpected:    req.GetCheckExpected(),
	}

	var result *usecase.ApplyResult
	if saga.IsDTMBranch(ctx) {
		err := saga.RunWithBarrier(ctx, s.db, func(tx *gorm.DB) error {
			r, err := s.applyUC.ApplyPromotionInTx(ctx, tx, ureq)
			if err != nil {
				return err
			}
			result = r
			if !r.Success {
				return saga.Abort(r.ErrorReason)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			// Barrier suppressed a duplicate/late call — report success with the
			// already-persisted breakdown (idempotent re-read).
			r, rerr := s.applyUC.ApplyPromotion(ctx, ureq)
			if rerr != nil {
				return nil, status.Error(codes.Internal, "apply promotion failed")
			}
			result = r
		}
		return applyResultToProto(result), nil
	}

	result, err := s.applyUC.ApplyPromotion(ctx, ureq)
	if err != nil {
		return nil, status.Error(codes.Internal, "apply promotion failed")
	}

	return applyResultToProto(result), nil
}

// applyResultToProto maps the usecase breakdown onto the wire response.
func applyResultToProto(result *usecase.ApplyResult) *promotionv1.ApplyPromotionResponse {
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
	}
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
	err := saga.RunWithBarrier(ctx, s.db, func(tx *gorm.DB) error {
		return s.applyUC.ConfirmUsageInTx(ctx, tx, req.GetOrderId())
	})
	if err != nil {
		if saga.IsDTMBranch(ctx) {
			return nil, err // let DTM retry with the real code
		}
		return nil, status.Error(codes.Internal, "confirm usage failed")
	}
	return &promotionv1.ConfirmUsageResponse{Success: true}, nil
}

// ReleaseUsage voids RESERVED usages and decrements used_count. Doubles as the
// compensation branch for both ApplyPromotion and ConfirmUsage in the DTM saga.
func (s *PromotionServiceServer) ReleaseUsage(ctx context.Context, req *promotionv1.ReleaseUsageRequest) (*promotionv1.ReleaseUsageResponse, error) {
	err := saga.RunWithBarrier(ctx, s.db, func(tx *gorm.DB) error {
		return s.applyUC.ReleaseUsageInTx(ctx, tx, req.GetOrderId())
	})
	if err != nil {
		if saga.IsDTMBranch(ctx) {
			return nil, err
		}
		return nil, status.Error(codes.Internal, "release usage failed")
	}
	return &promotionv1.ReleaseUsageResponse{Success: true}, nil
}
