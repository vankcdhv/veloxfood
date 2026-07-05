package usecase

import (
	"context"
	"encoding/json"
	"log/slog"

	"project/pkg/saga"
	paymentv1 "project/proto/payment/v1"
	promotionv1 "project/proto/promotion/v1"
	"project/services/order/internal/entity"
)

// driveCancelSaga synchronously drives the money/voucher legs of a
// cancellation through DTM: Payment.Refund (paid-online orders) and
// Promotion.ReleaseUsage (orders holding voucher reservations). Branches have
// no compensations — they are idempotent forward operations DTM retries to
// completion.
//
// The order.cancelled event still fans out to every consumer, and the payment
// and promotion consumers stay live as an idempotent safety net: a failure
// here only loses the *instant* refund, never the refund itself. The order's
// payment_status flips to REFUNDED exclusively via the payment.refunded event
// (payment stays the source of truth for money movements).
func (uc *orderLifecycleUsecase) driveCancelSaga(ctx context.Context, order *entity.Order) {
	if !uc.sagaCfg.DTMEnabled() {
		return
	}

	refund := order.PaymentStatus == entity.PaymentPaid && order.PaymentMethod != entity.MethodCOD
	var codes []string
	_ = json.Unmarshal(order.VoucherCodes, &codes)
	release := len(codes) > 0
	if !refund && !release {
		return
	}

	sg, err := saga.NewSaga(uc.sagaCfg.DTMAddr)
	if err != nil {
		slog.WarnContext(ctx, "cancel(dtm): coordinator unreachable — consumers will converge",
			"order_id", order.ID, "err", err)
		return
	}

	if refund {
		sg.Add(uc.sagaCfg.PaymentBranch+"/payment.v1.PaymentService/Refund", "", &paymentv1.RefundRequest{
			OrderId: order.ID,
			Amount:  order.GrandTotal,
		})
	}
	if release {
		sg.Add(uc.sagaCfg.PromotionBranch+"/promotion.v1.PromotionService/ReleaseUsage", "", &promotionv1.ReleaseUsageRequest{
			OrderId: order.ID,
		})
	}

	if err := sg.Submit(); err != nil {
		slog.WarnContext(ctx, "cancel(dtm): saga submit failed — consumers will converge",
			"order_id", order.ID, "err", err)
		return
	}
	slog.InfoContext(ctx, "cancel(dtm): refund/release driven synchronously",
		"order_id", order.ID, "gid", sg.Gid, "refund", refund, "release", release)
}
