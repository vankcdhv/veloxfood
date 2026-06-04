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
	CustomerID string
	StoreID    string
	LocationID string // empty for PICKUP
	// LocationLevel is the delivery granularity: "BUILDING" | "FLOOR" | "ROOM".
	// Empty / absent ⇒ treated as "ROOM" for back-compat. Only used for DELIVERY.
	LocationLevel string
	Fulfillment   entity.Fulfillment
	PaymentMethod entity.PaymentMethod
	VoucherCodes  []string
	Items         []PlaceOrderItem
	// DesiredTime is the customer's requested receive time (RFC3339). Empty = ASAP.
	DesiredTime string
}

// PlaceOrderItem is one line item from the customer.
type PlaceOrderItem struct {
	MenuItemID      string
	Qty             int
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
	cartRepo      repository.CartRepository
	outboxRepo    repository.OutboxRepository
	storeClient   *grpcclient.StoreClient
	promoClient   *grpcclient.PromotionClient
	paymentClient *grpcclient.PaymentClient
}

// NewPlaceOrderUsecase constructs the saga orchestrator.
func NewPlaceOrderUsecase(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	outboxRepo repository.OutboxRepository,
	storeClient *grpcclient.StoreClient,
	promoClient *grpcclient.PromotionClient,
	paymentClient *grpcclient.PaymentClient,
) PlaceOrderUsecase {
	return &placeOrderUsecase{
		db:            db,
		orderRepo:     orderRepo,
		cartRepo:      cartRepo,
		outboxRepo:    outboxRepo,
		storeClient:   storeClient,
		promoClient:   promoClient,
		paymentClient: paymentClient,
	}
}

// PlaceOrder executes the synchronous saga:
//  1. Validate store (open-now gate, desired_time window)
//  2. Resolve prices + ship_fee
//  3. Apply promotions (gRPC)
//  4. Capture payment (gRPC, WALLET/MOMO only)
//  5. Persist order + items + history + outbox in one DB tx
//
// Compensation on any failure (in reverse order):
// Payment.Refund → Promotion.ReleaseUsage
func (uc *placeOrderUsecase) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*PlaceOrderResult, error) {
	traceID := outbox.TraceIDFromCtx(ctx)

	// ── Step 1: Validate store ────────────────────────────────────────────────
	if uc.storeClient == nil {
		return nil, ErrStoreNotFound
	}
	storeInfo, err := uc.storeClient.GetStoreForOrder(ctx, req.StoreID, req.LocationID, req.LocationLevel)
	if err != nil {
		return nil, ErrStoreNotFound
	}
	if !storeInfo.Found {
		return nil, ErrStoreNotFound
	}
	// Server-authoritative open-now gate: reject if store is not currently open.
	if !storeInfo.OpenNow {
		return nil, ErrStoreClosed
	}
	if req.Fulfillment == entity.FulfillmentDelivery && !storeInfo.Served {
		return nil, ErrStoreNotServed
	}

	// ── Step 1b: Validate desired_time ────────────────────────────────────────
	// Rules (server authoritative):
	//   • If provided: must be today (local), >= now (grace: >= now, not now+prep),
	//     and <= close_time today.
	//   • If absent: treated as ASAP (null stored); no validation required.
	var desiredTimePtr *time.Time
	if req.DesiredTime != "" {
		dt, parseErr := time.Parse(time.RFC3339, req.DesiredTime)
		if parseErr != nil {
			return nil, ErrDesiredTimePast // bad format ⇒ treat as invalid
		}
		now := time.Now()

		// Must be the same local calendar day.
		if !isSameLocalDay(dt) {
			return nil, ErrDesiredTimeNotToday
		}

		// Must not be before now (small grace: we allow equal-to-now).
		if dt.Before(now) {
			return nil, ErrDesiredTimePast
		}

		// Must not exceed today's close time (if known).
		if storeInfo.CloseTimeToday != "" {
			closeTime, ok := parseCloseTime(storeInfo.CloseTimeToday)
			if ok && dt.After(closeTime) {
				return nil, ErrDesiredTimeAfterClose
			}
		}

		desiredTimePtr = &dt
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

		orderItems = append(orderItems, &entity.OrderItem{
			MenuItemID:      ri.MenuItemID,
			NameSnapshot:    storeItem.Name,
			PriceSnapshot:   storeItem.Price,
			Qty:             ri.Qty,
			OptionsSnapshot: opts,
		})
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

	// ── Step 4: Capture payment (WALLET/MOMO) ─────────────────────────────────
	var payURL string
	// WALLET capture is synchronous: CAPTURED ⇒ debited immediately ⇒ PAID.
	// MOMO returns PENDING + pay_url → stays UNPAID until payment.captured event.
	// COD stays UNPAID until delivered.
	paymentStatus := entity.PaymentUnpaid

	if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
		capResult, capErr := uc.paymentClient.Capture(ctx, orderID, req.CustomerID, grandTotal, string(req.PaymentMethod))
		if capErr != nil || capResult.Status == "FAILED" {
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

	// ── Step 5: Confirm promotion usages, then persist in one DB tx ───────────
	if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
		if confErr := uc.promoClient.ConfirmUsage(ctx, orderID); confErr != nil {
			slog.WarnContext(ctx, "place order: confirm usage failed — compensating", "err", confErr)
			if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
				_ = uc.paymentClient.Refund(ctx, orderID, grandTotal)
			}
			_ = uc.promoClient.ReleaseUsage(ctx, orderID)
			return nil, ErrPromotionInvalid
		}
	}

	codeDay := time.Now().UTC()
	codeSeq, seqErr := uc.orderRepo.NextDailyCodeSeq(ctx, codeDay)
	if seqErr != nil {
		if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
			_ = uc.paymentClient.Refund(ctx, orderID, grandTotal)
		}
		if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
			_ = uc.promoClient.ReleaseUsage(ctx, orderID)
		}
		return nil, seqErr
	}
	code := buildOrderCode(codeDay, codeSeq)

	var pickupPin *string
	if req.Fulfillment == entity.FulfillmentPickup {
		pin := generatePickupPIN()
		pickupPin = &pin
	}

	var locationIDPtr *string
	var locationLevelPtr *string
	if req.Fulfillment == entity.FulfillmentDelivery {
		if req.LocationID != "" {
			locationIDPtr = strPtr(req.LocationID)
		}
		// Default to ROOM when level is absent (back-compat with old clients).
		level := req.LocationLevel
		if level == "" {
			level = "ROOM"
		}
		locationLevelPtr = strPtr(level)
	}

	// Resolve effective location level for event payloads.
	effectiveLevel := req.LocationLevel
	if effectiveLevel == "" {
		effectiveLevel = "ROOM"
	}

	// desired_time in event payload: RFC3339 string or "" when ASAP.
	desiredTimeStr := ""
	if desiredTimePtr != nil {
		desiredTimeStr = desiredTimePtr.UTC().Format(time.RFC3339)
	}

	order := &entity.Order{
		ID:            orderID,
		Code:          code,
		CustomerID:    req.CustomerID,
		StoreID:       req.StoreID,
		LocationID:    locationIDPtr,
		LocationLevel: locationLevelPtr,
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
		DesiredTime:   desiredTimePtr,
		PlacedAt:      time.Now().UTC(),
	}

	// Build frozen order.placed payload.
	placedItems := make([]map[string]any, len(req.Items))
	for i, ri := range req.Items {
		storeItem := priceMap[ri.MenuItemID]
		placedItems[i] = map[string]any{
			"menu_item_id":   ri.MenuItemID,
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
		"location_level": effectiveLevel,
		"fulfillment":    string(req.Fulfillment),
		"payment_method": string(req.PaymentMethod),
		"items_total":    itemsTotal,
		"ship_fee":       shipFee,
		"discount":       discount,
		"grand_total":    grandTotal,
		"desired_time":   desiredTimeStr,
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
		if req.PaymentMethod != entity.MethodCOD && uc.paymentClient != nil {
			_ = uc.paymentClient.Refund(ctx, orderID, grandTotal)
		}
		if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
			_ = uc.promoClient.ReleaseUsage(ctx, orderID)
		}
		slog.ErrorContext(ctx, "place order: db tx failed", "order_id", orderID, "err", txErr)
		return nil, txErr
	}

	slog.InfoContext(ctx, "order placed",
		"order_id", orderID, "code", code,
		"grand_total", grandTotal, "desired_time", desiredTimeStr)

	// Best-effort: drop the ordered items from the customer's cart for this store
	// (leaving items from other stores intact). A failure here must not fail the
	// order — the order is already committed — so we only log.
	uc.removeOrderedItemsFromCart(ctx, req.CustomerID, req.StoreID, req.Items)

	return &PlaceOrderResult{
		OrderID:    orderID,
		Code:       code,
		GrandTotal: grandTotal,
		PayURL:     payURL,
	}, nil
}

// removeOrderedItemsFromCart deletes the just-ordered menu items from the
// customer's per-store cart. Best-effort: any error is logged, never returned.
func (uc *placeOrderUsecase) removeOrderedItemsFromCart(ctx context.Context, customerID, storeID string, items []PlaceOrderItem) {
	if uc.cartRepo == nil || len(items) == 0 {
		return
	}
	cart, err := uc.cartRepo.GetByCustomerAndStore(ctx, customerID, storeID)
	if err != nil || cart == nil {
		if err != nil {
			slog.WarnContext(ctx, "place order: load cart for cleanup failed", "err", err)
		}
		return
	}
	for _, it := range items {
		if rmErr := uc.cartRepo.RemoveItem(ctx, cart.ID, it.MenuItemID); rmErr != nil {
			slog.WarnContext(ctx, "place order: cart item cleanup failed",
				"cart_id", cart.ID, "menu_item_id", it.MenuItemID, "err", rmErr)
		}
	}
}

// apperrorPayment converts a payment capture error to a user-facing error.
// When the gateway reported FAILED without a transport error, surface a payment
// failure (not a misleading "store closed").
func apperrorPayment(err error) error {
	if err != nil {
		return err
	}
	return ErrPaymentFailed
}
