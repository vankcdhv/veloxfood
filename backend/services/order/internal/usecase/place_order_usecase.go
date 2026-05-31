package usecase

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"project/pkg/outbox"
	"project/services/order/internal/entity"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/repository"

	"gorm.io/gorm"
)

// PlaceOrderRequest is the input for the place-order saga.
type PlaceOrderRequest struct {
	CustomerID    string
	StoreID       string
	LocationID    string // empty for PICKUP
	Fulfillment   entity.Fulfillment
	PaymentMethod entity.PaymentMethod
	VoucherCodes  []string
	Items         []PlaceOrderItem
}

// PlaceOrderItem is one line item from the customer.
type PlaceOrderItem struct {
	MenuItemID      string
	Qty             int
	CutoffID        string
	Date            string // YYYY-MM-DD
	OptionsSnapshot json.RawMessage
}

// PlaceOrderResult is returned to the HTTP handler on success.
type PlaceOrderResult struct {
	OrderID    string
	Code       string
	GrandTotal int64
	PayURL     string // non-empty for MOMO
}

// PlaceOrderUsecase orchestrates the synchronous order-placement saga.
type PlaceOrderUsecase interface {
	PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*PlaceOrderResult, error)
}

type placeOrderUsecase struct {
	db            *gorm.DB
	orderRepo     repository.OrderRepository
	outboxRepo    repository.OutboxRepository
	storeClient   *grpcclient.StoreClient
	promoClient   *grpcclient.PromotionClient
	paymentClient *grpcclient.PaymentClient
}

// NewPlaceOrderUsecase constructs the saga orchestrator.
func NewPlaceOrderUsecase(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	outboxRepo repository.OutboxRepository,
	storeClient *grpcclient.StoreClient,
	promoClient *grpcclient.PromotionClient,
	paymentClient *grpcclient.PaymentClient,
) PlaceOrderUsecase {
	return &placeOrderUsecase{
		db:            db,
		orderRepo:     orderRepo,
		outboxRepo:    outboxRepo,
		storeClient:   storeClient,
		promoClient:   promoClient,
		paymentClient: paymentClient,
	}
}

// PlaceOrder executes the synchronous saga:
//  1. Validate store (OPEN, deadline, served)
//  2. Resolve prices + ship_fee
//  3. Apply promotions (gRPC)
//  4. Decrement slot quota per item (gRPC)
//  5. Capture payment (gRPC, WALLET/MOMO only)
//  6. Persist order + items + history + outbox in one DB tx
//
// Compensation on any failure (in reverse order):
// Payment.Refund → Promotion.ReleaseUsage → Store.RestoreSlotQuota
func (uc *placeOrderUsecase) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*PlaceOrderResult, error) {
	traceID := outbox.TraceIDFromCtx(ctx)

	// ── Step 1: Validate store ────────────────────────────────────────────────
	roomID := req.LocationID
	storeInfo, err := uc.storeClient.GetStoreForOrder(ctx, req.StoreID, roomID)
	if err != nil {
		return nil, ErrStoreNotFound
	}
	if !storeInfo.Found {
		return nil, ErrStoreNotFound
	}
	if storeInfo.SaleStatus != "OPEN" {
		return nil, ErrStoreClosed
	}
	if req.Fulfillment == entity.FulfillmentDelivery && !storeInfo.Served {
		return nil, ErrStoreNotServed
	}
	// storeInfo.OrderDeadline is "today's earliest cutoff − lead". It only gates
	// same-day orders; a pre-order for a future date is bounded by the ≤7-day rule
	// (Step 3), not by a cutoff that already passed today.
	ordersForToday := false
	todayStr := time.Now().Format("2006-01-02")
	for _, ri := range req.Items {
		if ri.Date == "" || ri.Date == todayStr {
			ordersForToday = true
			break
		}
	}
	if ordersForToday && storeInfo.OrderDeadline != "" {
		deadline, parseErr := time.Parse(time.RFC3339, storeInfo.OrderDeadline)
		if parseErr == nil && time.Now().UTC().After(deadline) {
			return nil, ErrOrderDeadlinePassed
		}
	}

	// ── Step 2: Resolve prices + ship fee ─────────────────────────────────────
	priceMap := make(map[string]grpcclient.StoreOrderItem, len(storeInfo.Items))
	for _, it := range storeInfo.Items {
		priceMap[it.ItemID] = it
	}

	var itemsTotal int64
	orderItems := make([]*entity.OrderItem, 0, len(req.Items))
	for _, ri := range req.Items {
		storeItem, ok := priceMap[ri.MenuItemID]
		if !ok {
			return nil, ErrItemNotInStore
		}
		lineTotal := storeItem.Price * int64(ri.Qty)
		itemsTotal += lineTotal

		opts := ri.OptionsSnapshot
		if opts == nil {
			opts = json.RawMessage("[]")
		}

		var datePtr *time.Time
		if ri.Date != "" {
			if t, parseErr := parseDate(ri.Date); parseErr == nil {
				// Pre-order: enforce ≤ 7 days.
				if t.After(time.Now().UTC().Add(7 * 24 * time.Hour)) {
					return nil, ErrPreOrderTooFar
				}
				datePtr = &t
			}
		}

		oi := &entity.OrderItem{
			MenuItemID:      ri.MenuItemID,
			NameSnapshot:    storeItem.Name,
			PriceSnapshot:   storeItem.Price,
			Qty:             ri.Qty,
			OptionsSnapshot: opts,
		}
		if ri.CutoffID != "" {
			oi.CutoffID = strPtr(ri.CutoffID)
		}
		if datePtr != nil {
			oi.Date = datePtr
		}
		orderItems = append(orderItems, oi)
	}

	// Ship fee: unit_fee × total_qty for DELIVERY; 0 for PICKUP.
	var shipFee int64
	if req.Fulfillment == entity.FulfillmentDelivery {
		var totalQty int64
		for _, ri := range req.Items {
			totalQty += int64(ri.Qty)
		}
		shipFee = storeInfo.UnitShipFee * totalQty
	}

	// ── Step 3: Apply promotions ──────────────────────────────────────────────
	orderID := outbox.NewEventID() // pre-generate so saga steps share it
	var discount int64

	if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
		var totalItems int32
		for _, ri := range req.Items {
			totalItems += int32(ri.Qty)
		}
		promoResult, promoErr := uc.promoClient.ApplyPromotion(
			ctx, orderID, req.StoreID, req.CustomerID,
			req.VoucherCodes, itemsTotal, totalItems,
		)
		if promoErr != nil {
			slog.WarnContext(ctx, "place order: promotion apply failed", "err", promoErr)
			return nil, ErrPromotionInvalid
		}
		if !promoResult.Success {
			return nil, ErrPromotionInvalid
		}
		discount = promoResult.ItemDiscount + promoResult.ShipDiscount
	}

	grandTotal := itemsTotal + shipFee - discount
	if grandTotal < 0 {
		grandTotal = 0
	}

	// ── Step 4: Decrement slot quota per item ─────────────────────────────────
	decremented := make([]PlaceOrderItem, 0, len(req.Items))
	for _, ri := range req.Items {
		if ri.CutoffID == "" || ri.Date == "" {
			continue // no quota slot — skip
		}
		ok, qErr := uc.storeClient.DecrementSlotQuota(ctx, ri.MenuItemID, ri.CutoffID, ri.Date, ri.Qty)
		if qErr != nil || !ok {
			// Compensation: restore already-decremented slots.
			uc.compensateQuota(ctx, decremented)
			if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
				_ = uc.promoClient.ReleaseUsage(ctx, orderID)
			}
			return nil, ErrQuotaExceeded
		}
		decremented = append(decremented, ri)
	}

	// ── Step 5: Capture payment (WALLET/MOMO) ─────────────────────────────────
	var payURL string
	// WALLET capture is synchronous: a CAPTURED result means the wallet was
	// already debited, so the order is PAID immediately (no need to wait for the
	// payment.captured event). MOMO returns PENDING + pay_url → stays UNPAID until
	// the IPN-driven payment.captured event arrives. COD stays UNPAID until delivered.
	paymentStatus := entity.PaymentUnpaid

	if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
		capResult, capErr := uc.paymentClient.Capture(ctx, orderID, req.CustomerID, grandTotal, string(req.PaymentMethod))
		if capErr != nil || capResult.Status == "FAILED" {
			uc.compensateQuota(ctx, decremented)
			if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
				_ = uc.promoClient.ReleaseUsage(ctx, orderID)
			}
			return nil, apperrorPayment(capErr)
		}
		payURL = capResult.PayURL
		if capResult.Status == "CAPTURED" {
			paymentStatus = entity.PaymentPaid
		}
	}

	// ── Step 6: Confirm promotion usages, then persist in one DB tx ───────────
	if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
		if confErr := uc.promoClient.ConfirmUsage(ctx, orderID); confErr != nil {
			slog.WarnContext(ctx, "place order: confirm usage failed — compensating", "err", confErr)
			if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
				_ = uc.paymentClient.Refund(ctx, orderID, grandTotal)
			}
			uc.compensateQuota(ctx, decremented)
			_ = uc.promoClient.ReleaseUsage(ctx, orderID)
			return nil, ErrPromotionInvalid
		}
	}

	code := generateOrderCode()
	var pickupPin *string
	if req.Fulfillment == entity.FulfillmentPickup {
		pin := generatePickupPIN()
		pickupPin = &pin
	}

	var locationIDPtr *string
	if req.LocationID != "" && req.Fulfillment == entity.FulfillmentDelivery {
		locationIDPtr = strPtr(req.LocationID)
	}

	order := &entity.Order{
		ID:            orderID,
		Code:          code,
		CustomerID:    req.CustomerID,
		StoreID:       req.StoreID,
		LocationID:    locationIDPtr,
		Fulfillment:   req.Fulfillment,
		Status:        entity.StatusPending,
		ItemsTotal:    itemsTotal,
		ShipFee:       shipFee,
		Discount:      discount,
		GrandTotal:    grandTotal,
		PaymentMethod: req.PaymentMethod,
		PaymentStatus: paymentStatus,
		VoucherCodes:  codesJSON(req.VoucherCodes),
		PickupPin:     pickupPin,
		PlacedAt:      time.Now().UTC(),
	}

	// Build frozen order.placed payload (§2bis).
	placedItems := make([]map[string]any, len(req.Items))
	for i, ri := range req.Items {
		storeItem := priceMap[ri.MenuItemID]
		placedItems[i] = map[string]any{
			"menu_item_id":   ri.MenuItemID,
			"cutoff_id":      ri.CutoffID,
			"date":           ri.Date,
			"qty":            ri.Qty,
			"name_snapshot":  storeItem.Name,
			"price_snapshot": storeItem.Price,
		}
	}
	placedPayload, _ := json.Marshal(map[string]any{
		"order_id":       orderID,
		"code":           code,
		"customer_id":    req.CustomerID,
		"store_id":       req.StoreID,
		"location_id":    req.LocationID,
		"fulfillment":    string(req.Fulfillment),
		"payment_method": string(req.PaymentMethod),
		"items_total":    itemsTotal,
		"ship_fee":       shipFee,
		"discount":       discount,
		"grand_total":    grandTotal,
		"items":          placedItems,
	})

	outboxEvt := &entity.OutboxEvent{
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     "order.placed",
		Payload:       placedPayload,
		TraceID:       strPtr(traceID),
	}

	histNote := strPtr("order placed")
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := uc.orderRepo.Create(ctx, tx, order, orderItems); err != nil {
			return err
		}
		if err := uc.orderRepo.UpdateStatus(ctx, tx, orderID, entity.StatusPending, nil, histNote); err != nil {
			return err
		}
		return uc.outboxRepo.Append(ctx, tx, outboxEvt)
	})
	if txErr != nil {
		// Compensate all saga steps — order was not persisted.
		if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
			_ = uc.paymentClient.Refund(ctx, orderID, grandTotal)
		}
		uc.compensateQuota(ctx, decremented)
		if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
			_ = uc.promoClient.ReleaseUsage(ctx, orderID)
		}
		slog.ErrorContext(ctx, "place order: db tx failed", "order_id", orderID, "err", txErr)
		return nil, txErr
	}

	slog.InfoContext(ctx, "order placed", "order_id", orderID, "code", code, "grand_total", grandTotal)
	return &PlaceOrderResult{
		OrderID:    orderID,
		Code:       code,
		GrandTotal: grandTotal,
		PayURL:     payURL,
	}, nil
}

// compensateQuota restores slot quotas for all items that were successfully
// decremented before a saga failure. Errors are logged but not returned —
// this is a best-effort compensation; eventual consistency is acceptable here.
func (uc *placeOrderUsecase) compensateQuota(ctx context.Context, decremented []PlaceOrderItem) {
	for _, ri := range decremented {
		if ri.CutoffID == "" || ri.Date == "" {
			continue
		}
		if err := uc.storeClient.RestoreSlotQuota(ctx, ri.MenuItemID, ri.CutoffID, ri.Date, ri.Qty); err != nil {
			slog.ErrorContext(ctx, "compensate: restore quota failed",
				"menu_item_id", ri.MenuItemID, "err", err)
		}
	}
}

// apperrorPayment converts a payment capture error to a user-facing apperror.
func apperrorPayment(err error) error {
	if err != nil {
		return err
	}
	return ErrStoreClosed // fallback — caller checks capResult.Status == FAILED
}
