package grpcclient

import (
	"context"
	"fmt"

	"project/pkg/grpcx"
	paymentv1 "project/proto/payment/v1"
)

// PaymentCaptureResult is the outcome of Payment.Capture.
type PaymentCaptureResult struct {
	PaymentID string
	Status    string // CAPTURED | PENDING | FAILED
	PayURL    string // non-empty for MOMO
	Error     string // non-empty on failure
}

// PaymentClient wraps the Payment gRPC service for use by the Order saga.
type PaymentClient struct {
	client paymentv1.PaymentServiceClient
}

// NewPaymentClient dials the payment service.
func NewPaymentClient(addr string) (*PaymentClient, error) {
	conn, err := grpcx.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("payment grpc client: dial %s: %w", addr, err)
	}
	return &PaymentClient{client: paymentv1.NewPaymentServiceClient(conn)}, nil
}

// Capture authorises or initiates payment for an order.
// WALLET → sync debit, returns CAPTURED.
// MOMO   → creates intent, returns PENDING + PayURL.
// COD    → no-op, returns PENDING.
func (c *PaymentClient) Capture(ctx context.Context, orderID, customerID string, amount int64, method string) (*PaymentCaptureResult, error) {
	resp, err := c.client.Capture(ctx, &paymentv1.CaptureRequest{
		OrderId:    orderID,
		CustomerId: customerID,
		Amount:     amount,
		Method:     method,
	})
	if err != nil {
		return nil, fmt.Errorf("payment.Capture: %w", err)
	}
	return &PaymentCaptureResult{
		PaymentID: resp.GetPaymentId(),
		Status:    resp.GetStatus(),
		PayURL:    resp.GetPayUrl(),
		Error:     resp.GetError(),
	}, nil
}

// PaymentStatusResult is the outcome of GetPaymentStatus.
type PaymentStatusResult struct {
	Status string // UNPAID | PAID | REFUNDED
	Method string
	Amount int64
	PayURL string // MoMo checkout link persisted at capture time
}

// GetPaymentStatus reads the payment state for an order — the DTM place-order
// path uses it to fetch the MoMo pay_url after the saga completes (branch
// responses are not forwarded to the saga opener).
func (c *PaymentClient) GetPaymentStatus(ctx context.Context, orderID string) (*PaymentStatusResult, error) {
	resp, err := c.client.GetPaymentStatus(ctx, &paymentv1.GetPaymentStatusRequest{OrderId: orderID})
	if err != nil {
		return nil, fmt.Errorf("payment.GetPaymentStatus: %w", err)
	}
	return &PaymentStatusResult{
		Status: resp.GetStatus(),
		Method: resp.GetMethod(),
		Amount: resp.GetAmount(),
		PayURL: resp.GetPayUrl(),
	}, nil
}

// Refund credits 100% of the captured amount back to the customer wallet.
// Idempotent by order_id.
func (c *PaymentClient) Refund(ctx context.Context, orderID string, amount int64) error {
	resp, err := c.client.Refund(ctx, &paymentv1.RefundRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return fmt.Errorf("payment.Refund: %w", err)
	}
	if !resp.GetSuccess() {
		return fmt.Errorf("payment.Refund returned success=false")
	}
	return nil
}
