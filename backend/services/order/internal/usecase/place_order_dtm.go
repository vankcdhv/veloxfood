package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"project/pkg/outbox"
	"project/pkg/saga"
	paymentv1 "project/proto/payment/v1"
	promotionv1 "project/proto/promotion/v1"
	"project/services/order/internal/entity"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// placeViaDTM runs placement through the DTM saga coordinator.
//
// DTM branches take static payloads fixed at submit time, so the price must be
// final before the saga opens: the promotion discount is QUOTED first
// (read-only), and the ApplyPromotion branch re-checks the reservation against
// that quote — drift aborts the saga. Branch responses are not forwarded to
// the opener, so the MoMo pay_url is read back via GetPaymentStatus after the
// saga completes. The order row persists locally only on saga success; a
// persistence failure re-uses the pending_compensations machinery.
//
// Branch layout (compensations run in reverse on failure):
//  1. Promotion.ApplyPromotion  / Promotion.ReleaseUsage
//  2. Payment.Capture           / Payment.Refund        (skipped for COD)
//  3. Promotion.ConfirmUsage    / Promotion.ReleaseUsage
func (uc *placeOrderUsecase) placeViaDTM(ctx context.Context, req PlaceOrderRequest, d *orderDraft) (*PlaceOrderResult, error) {
	orderID := outbox.NewEventID()
	hasVouchers := len(req.VoucherCodes) > 0
	needCapture := req.PaymentMethod != entity.MethodCOD

	if hasVouchers && uc.promoClient == nil {
		return nil, ErrPromotionInvalid
	}
	if needCapture && uc.paymentClient == nil {
		return nil, ErrPaymentFailed
	}

	// ── Quote the discount before the saga so the capture amount is final ────
	var discount int64
	if hasVouchers {
		quote, err := uc.promoClient.QuotePromotion(ctx, req.StoreID, req.CustomerID, req.VoucherCodes, d.itemsTotal, d.totalQty)
		if err != nil {
			slog.WarnContext(ctx, "place order(dtm): promotion quote failed", "err", err)
			return nil, ErrPromotionInvalid
		}
		if !quote.Success {
			return nil, ErrPromotionInvalid
		}
		discount = quote.ItemDiscount + quote.ShipDiscount
	}

	grandTotal := d.itemsTotal + d.shipFee - discount
	if grandTotal < 0 {
		grandTotal = 0
	}

	// ── Open and submit the saga (WaitResult: blocks until terminal state) ───
	if hasVouchers || needCapture {
		if err := uc.submitPlacementSaga(ctx, req, d, orderID, discount, grandTotal, hasVouchers, needCapture); err != nil {
			return nil, err
		}
	}
	// COD without vouchers has nothing to coordinate — persist directly.

	// ── Resolve payment outcome (branch responses aren't forwarded) ──────────
	paymentStatus := entity.PaymentUnpaid
	var payURL string
	if needCapture {
		if st, err := uc.paymentClient.GetPaymentStatus(ctx, orderID); err == nil {
			if st.Status == "PAID" {
				paymentStatus = entity.PaymentPaid
			}
			payURL = st.PayURL
		} else if req.PaymentMethod == entity.MethodWallet {
			// The saga is green, so the wallet WAS captured — a read blip must
			// not misreport the order as unpaid.
			slog.WarnContext(ctx, "place order(dtm): payment status read failed — deriving PAID from saga success", "err", err)
			paymentStatus = entity.PaymentPaid
		} else {
			// MoMo without a pay_url is unusable for the customer: roll back.
			slog.ErrorContext(ctx, "place order(dtm): cannot read pay_url — compensating", "order_id", orderID, "err", err)
			uc.compensate(ctx, orderID, grandTotal, true, hasVouchers)
			return nil, ErrPaymentFailed
		}
	}

	return uc.persistPlacedOrder(ctx, req, d, orderID, discount, grandTotal, paymentStatus, payURL)
}

// submitPlacementSaga builds the branch list and submits it to DTM.
func (uc *placeOrderUsecase) submitPlacementSaga(
	ctx context.Context, req PlaceOrderRequest, d *orderDraft,
	orderID string, discount, grandTotal int64,
	hasVouchers, needCapture bool,
) error {
	sg, err := saga.NewSaga(uc.sagaCfg.DTMAddr)
	if err != nil {
		return fmt.Errorf("%w: %v", errDTMUnavailable, err)
	}

	promo := uc.sagaCfg.PromotionBranch + "/promotion.v1.PromotionService/"
	pay := uc.sagaCfg.PaymentBranch + "/payment.v1.PaymentService/"

	if hasVouchers {
		sg.Add(promo+"ApplyPromotion", promo+"ReleaseUsage", &promotionv1.ApplyPromotionRequest{
			OrderId:          orderID,
			StoreId:          req.StoreID,
			CustomerId:       req.CustomerID,
			Codes:            req.VoucherCodes,
			Subtotal:         d.itemsTotal, // same input the quote priced on
			ItemCount:        d.totalQty,
			ExpectedDiscount: discount,
			CheckExpected:    true,
		})
	}
	if needCapture {
		sg.Add(pay+"Capture", pay+"Refund", &paymentv1.CaptureRequest{
			OrderId:    orderID,
			CustomerId: req.CustomerID,
			Amount:     grandTotal,
			Method:     string(req.PaymentMethod),
		})
	}
	if hasVouchers {
		sg.Add(promo+"ConfirmUsage", promo+"ReleaseUsage", &promotionv1.ConfirmUsageRequest{
			OrderId: orderID,
		})
	}

	if err := sg.Submit(); err != nil {
		return uc.mapSagaSubmitError(ctx, err, orderID, grandTotal, needCapture, hasVouchers)
	}
	slog.InfoContext(ctx, "place order(dtm): saga completed", "order_id", orderID, "gid", sg.Gid)
	return nil
}

// mapSagaSubmitError translates a WaitResult submit failure.
//   - Aborted: a branch rejected the business operation and DTM already ran the
//     compensations — map the reason to a user-facing error.
//   - anything else: transport/unknown. The saga may still be live inside DTM
//     (it keeps driving branches), so schedule local compensations as a durable
//     backstop — every compensation is idempotent and no-ops when DTM already
//     completed or rolled back the work.
func (uc *placeOrderUsecase) mapSagaSubmitError(
	ctx context.Context, err error,
	orderID string, grandTotal int64, refund, release bool,
) error {
	if st, ok := status.FromError(err); ok && st.Code() == codes.Aborted {
		msg := st.Message()
		slog.WarnContext(ctx, "place order(dtm): saga rolled back", "order_id", orderID, "reason", msg)
		switch {
		case strings.Contains(msg, "balance"):
			return ErrPaymentFailed
		case strings.Contains(msg, "promotion"), strings.Contains(msg, "discount"):
			return ErrPromotionInvalid
		default:
			return ErrPaymentFailed
		}
	}

	slog.ErrorContext(ctx, "place order(dtm): submit failed with unknown outcome — scheduling compensations",
		"order_id", orderID, "err", err)
	uc.compensate(ctx, orderID, grandTotal, refund, release)
	return fmt.Errorf("place order saga: %w", err)
}
