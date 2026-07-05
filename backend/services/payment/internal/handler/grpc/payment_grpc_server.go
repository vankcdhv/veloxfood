package grpc

import (
	"context"
	"errors"
	"log/slog"

	"project/pkg/saga"
	paymentv1 "project/proto/payment/v1"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// PaymentServiceServer implements the proto-generated gRPC PaymentService.
// Capture/Refund double as DTM saga branches: calls carrying DTM branch
// metadata run under the sub-transaction barrier and signal business failure
// via codes.Aborted; direct calls keep the body-level result contract.
type PaymentServiceServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	db        *gorm.DB
	captureUC usecase.CaptureUsecase
	refundUC  usecase.RefundUsecase
	paymentRepo interface {
		GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)
	}
}

func NewPaymentServiceServer(
	db *gorm.DB,
	captureUC usecase.CaptureUsecase,
	refundUC usecase.RefundUsecase,
	paymentRepo interface {
		GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)
	},
) *PaymentServiceServer {
	return &PaymentServiceServer{
		db:          db,
		captureUC:   captureUC,
		refundUC:    refundUC,
		paymentRepo: paymentRepo,
	}
}

// Capture authorises or initiates payment for the order saga.
// Idempotent by order_id.
func (s *PaymentServiceServer) Capture(ctx context.Context, req *paymentv1.CaptureRequest) (*paymentv1.CaptureResponse, error) {
	ureq := usecase.CaptureRequest{
		OrderID:    req.GetOrderId(),
		CustomerID: req.GetCustomerId(),
		Amount:     req.GetAmount(),
		Method:     entity.PaymentMethod(req.GetMethod()),
	}

	if saga.IsDTMBranch(ctx) {
		return s.captureAsBranch(ctx, ureq)
	}

	result, err := s.captureUC.Capture(ctx, ureq)
	if err != nil {
		slog.ErrorContext(ctx, "grpc: Capture failed", "order_id", req.GetOrderId(), "err", err)
		return nil, status.Error(codes.Internal, "capture failed")
	}
	return captureResultToProto(result), nil
}

// captureAsBranch runs the capture under the DTM barrier. An insufficient
// wallet balance aborts the saga (business failure); infrastructure errors
// stay retryable. MoMo gateway calls happen after the barrier transaction
// commits — a duplicate-suppressed call still finishes a dangling intent.
func (s *PaymentServiceServer) captureAsBranch(ctx context.Context, ureq usecase.CaptureRequest) (*paymentv1.CaptureResponse, error) {
	var result *usecase.CaptureResult
	err := saga.RunWithBarrier(ctx, s.db, func(tx *gorm.DB) error {
		r, err := s.captureUC.CaptureInTx(ctx, tx, ureq)
		if err != nil {
			if errors.Is(err, usecase.ErrInsufficientBalance) {
				return saga.Abort(err.Error())
			}
			return err
		}
		result = r
		return nil
	})
	if err != nil {
		return nil, err
	}

	if result == nil {
		// Barrier suppressed a duplicate call — re-read the persisted payment.
		r, rerr := s.captureUC.Capture(ctx, ureq)
		if rerr != nil {
			return nil, status.Error(codes.Internal, "capture failed")
		}
		result = r
	}
	if ureq.Method == entity.MethodMoMo && result.Status == entity.PaymentPending && result.PayURL == "" {
		r, ferr := s.captureUC.FinishMoMoIntent(ctx, result.PaymentID, ureq)
		if ferr != nil {
			// Gateway blip: leave the intent PENDING and let DTM retry the branch.
			return nil, status.Error(codes.Unavailable, "momo gateway unavailable")
		}
		result = r
	}
	if result.Status == entity.PaymentFailed {
		return nil, saga.Abort(result.Error)
	}
	return captureResultToProto(result), nil
}

func captureResultToProto(result *usecase.CaptureResult) *paymentv1.CaptureResponse {
	return &paymentv1.CaptureResponse{
		PaymentId: result.PaymentID,
		Status:    string(result.Status),
		PayUrl:    result.PayURL,
		Error:     result.Error,
	}
}

// Refund credits 100% back to the customer wallet. Idempotent by order_id.
// Doubles as the DTM compensation branch for Capture.
func (s *PaymentServiceServer) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
	if saga.IsDTMBranch(ctx) {
		var result *usecase.RefundResult
		err := saga.RunWithBarrier(ctx, s.db, func(tx *gorm.DB) error {
			r, err := s.refundUC.RefundInTx(ctx, tx, req.GetOrderId(), req.GetAmount())
			if err != nil {
				return err
			}
			result = r
			return nil
		})
		if err != nil {
			return nil, err
		}
		if result == nil {
			// Duplicate/null compensation suppressed by the barrier.
			result = &usecase.RefundResult{Success: true}
		}
		return &paymentv1.RefundResponse{Success: result.Success, RefundId: result.RefundID}, nil
	}

	result, err := s.refundUC.Refund(ctx, req.GetOrderId(), req.GetAmount())
	if err != nil {
		slog.ErrorContext(ctx, "grpc: Refund failed", "order_id", req.GetOrderId(), "err", err)
		return nil, status.Error(codes.Internal, "refund failed")
	}
	return &paymentv1.RefundResponse{
		Success:  result.Success,
		RefundId: result.RefundID,
	}, nil
}

// GetPaymentStatus returns the current payment state for an order.
func (s *PaymentServiceServer) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.GetPaymentStatusResponse, error) {
	p, err := s.paymentRepo.GetByOrderID(ctx, req.GetOrderId())
	if err != nil {
		return nil, status.Error(codes.Internal, "query failed")
	}
	if p == nil {
		return &paymentv1.GetPaymentStatusResponse{Status: "UNPAID"}, nil
	}

	st := "UNPAID"
	switch p.Status {
	case entity.PaymentCaptured:
		st = "PAID"
	case entity.PaymentRefunded:
		st = "REFUNDED"
	}

	return &paymentv1.GetPaymentStatusResponse{
		Status: st,
		Method: string(p.Method),
		Amount: p.Amount,
		PayUrl: p.PayURL,
	}, nil
}
