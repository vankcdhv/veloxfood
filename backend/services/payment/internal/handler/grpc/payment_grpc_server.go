package grpc

import (
	"context"
	"log/slog"

	paymentv1 "project/proto/payment/v1"
	"project/services/payment/internal/entity"
	"project/services/payment/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PaymentServiceServer implements the proto-generated gRPC PaymentService.
type PaymentServiceServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	captureUC usecase.CaptureUsecase
	refundUC  usecase.RefundUsecase
	paymentRepo interface {
		GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)
	}
}

func NewPaymentServiceServer(
	captureUC usecase.CaptureUsecase,
	refundUC usecase.RefundUsecase,
	paymentRepo interface {
		GetByOrderID(ctx context.Context, orderID string) (*entity.Payment, error)
	},
) *PaymentServiceServer {
	return &PaymentServiceServer{
		captureUC:   captureUC,
		refundUC:    refundUC,
		paymentRepo: paymentRepo,
	}
}

// Capture authorises or initiates payment for the order saga.
// Idempotent by order_id.
func (s *PaymentServiceServer) Capture(ctx context.Context, req *paymentv1.CaptureRequest) (*paymentv1.CaptureResponse, error) {
	method := entity.PaymentMethod(req.GetMethod())
	result, err := s.captureUC.Capture(ctx, usecase.CaptureRequest{
		OrderID:    req.GetOrderId(),
		CustomerID: req.GetCustomerId(),
		Amount:     req.GetAmount(),
		Method:     method,
	})
	if err != nil {
		slog.ErrorContext(ctx, "grpc: Capture failed", "order_id", req.GetOrderId(), "err", err)
		return nil, status.Error(codes.Internal, "capture failed")
	}
	return &paymentv1.CaptureResponse{
		PaymentId: result.PaymentID,
		Status:    string(result.Status),
		PayUrl:    result.PayURL,
		Error:     result.Error,
	}, nil
}

// Refund credits 100% back to the customer wallet. Idempotent by order_id.
func (s *PaymentServiceServer) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
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
