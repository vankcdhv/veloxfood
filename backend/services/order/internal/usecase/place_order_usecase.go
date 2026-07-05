package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"project/pkg/outbox"
	"project/pkg/saga"
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

// Saga-participant gateways. The concrete grpcclient types satisfy these;
// tests inject stubs to exercise the compensation branches.
type StoreGateway interface {
	GetStoreForOrder(ctx context.Context, storeID, roomID, locationLevel string) (*grpcclient.StoreForOrderResult, error)
}

type PromotionGateway interface {
	ApplyPromotion(ctx context.Context, orderID, storeID, customerID string, codes []string, subtotal int64, itemCount int32) (*grpcclient.PromotionApplyResult, error)
	QuotePromotion(ctx context.Context, storeID, customerID string, codes []string, subtotal int64, itemCount int32) (*grpcclient.PromotionApplyResult, error)
	ConfirmUsage(ctx context.Context, orderID string) error
	ReleaseUsage(ctx context.Context, orderID string) error
}

type PaymentGateway interface {
	Capture(ctx context.Context, orderID, customerID string, amount int64, method string) (*grpcclient.PaymentCaptureResult, error)
	Refund(ctx context.Context, orderID string, amount int64) error
	GetPaymentStatus(ctx context.Context, orderID string) (*grpcclient.PaymentStatusResult, error)
}

// SagaSettings selects the saga engine and addresses the DTM coordinator.
// Branch addresses are the participants' gRPC endpoints as reachable FROM the
// DTM server (a container): host.docker.internal:PORT in dev, compose service
// names in docker. Empty settings disable the DTM path entirely.
type SagaSettings struct {
	Engine          string // "dtm" (default) | "inline"
	DTMAddr         string // DTM server gRPC address
	PromotionBranch string
	PaymentBranch   string
}

type placeOrderUsecase struct {
	db            *gorm.DB
	orderRepo     repository.OrderRepository
	cartRepo      repository.CartRepository
	outboxRepo    repository.OutboxRepository
	compRepo      repository.CompensationRepository
	storeClient   StoreGateway
	promoClient   PromotionGateway
	paymentClient PaymentGateway
	sagaCfg       SagaSettings
}

// NewPlaceOrderUsecase constructs the saga orchestrator. Gateway params are
// interfaces — pass a true nil interface (not a typed-nil pointer) when a
// downstream client failed to dial.
func NewPlaceOrderUsecase(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	outboxRepo repository.OutboxRepository,
	compRepo repository.CompensationRepository,
	storeClient StoreGateway,
	promoClient PromotionGateway,
	paymentClient PaymentGateway,
	sagaCfg SagaSettings,
) PlaceOrderUsecase {
	return &placeOrderUsecase{
		db:            db,
		orderRepo:     orderRepo,
		cartRepo:      cartRepo,
		outboxRepo:    outboxRepo,
		compRepo:      compRepo,
		storeClient:   storeClient,
		promoClient:   promoClient,
		paymentClient: paymentClient,
		sagaCfg:       sagaCfg,
	}
}

// orderDraft carries the validated, priced inputs shared by both saga engines.
type orderDraft struct {
	priceMap    map[string]grpcclient.StoreOrderItem
	orderItems  []*entity.OrderItem
	itemsTotal  int64
	shipFee     int64
	desiredTime *time.Time
	totalQty    int32
}

// errDTMUnavailable marks a DTM path that never got off the ground (server
// unreachable before anything was submitted) — safe to fall back to inline.
var errDTMUnavailable = errors.New("dtm coordinator unavailable")

// PlaceOrder validates and prices the request, then runs the placement saga on
// the configured engine:
//   - dtm:    the DTM server coordinates the branches (automatic retry,
//     reverse compensation, sub-transaction barrier); the order row
//     persists locally after the saga succeeds.
//   - inline: hand-rolled in-process orchestration, kept as fallback and for
//     failure-injection comparison.
//
// If DTM is selected but unreachable before anything was submitted, placement
// falls back to inline — an order must never fail only because the
// coordinator is down.
func (uc *placeOrderUsecase) PlaceOrder(ctx context.Context, req PlaceOrderRequest) (*PlaceOrderResult, error) {
	draft, err := uc.prepareDraft(ctx, req)
	if err != nil {
		return nil, err
	}

	if uc.dtmEnabled() {
		res, dtmErr := uc.placeViaDTM(ctx, req, draft)
		if !errors.Is(dtmErr, errDTMUnavailable) {
			return res, dtmErr
		}
		slog.ErrorContext(ctx, "place order: DTM unreachable — falling back to inline saga", "err", dtmErr)
	}
	return uc.placeInline(ctx, req, draft)
}

// dtmEnabled reports whether the DTM engine is selected AND fully addressed.
func (uc *placeOrderUsecase) dtmEnabled() bool {
	return saga.ResolveEngine(uc.sagaCfg.Engine) == saga.EngineDTM &&
		uc.sagaCfg.DTMAddr != "" &&
		uc.sagaCfg.PromotionBranch != "" &&
		uc.sagaCfg.PaymentBranch != ""
}

// prepareDraft runs the read-only part of placement:
//  1. Validate store (open-now gate, desired_time window)
//  2. Resolve prices + ship_fee
func (uc *placeOrderUsecase) prepareDraft(ctx context.Context, req PlaceOrderRequest) (*orderDraft, error) {
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

	// Ship fee is a flat per-delivery charge for the resolved location (matches
	// the checkout preview); PICKUP pays nothing. It does not scale with quantity.
	var shipFee int64
	if req.Fulfillment == entity.FulfillmentDelivery {
		shipFee = storeInfo.UnitShipFee
	}

	var totalQty int32
	for _, ri := range req.Items {
		totalQty += int32(ri.Qty)
	}

	return &orderDraft{
		priceMap:    priceMap,
		orderItems:  orderItems,
		itemsTotal:  itemsTotal,
		shipFee:     shipFee,
		desiredTime: desiredTimePtr,
		totalQty:    totalQty,
	}, nil
}

// placeInline executes the hand-rolled synchronous saga:
//  3. Apply promotions (gRPC)
//  4. Capture payment (gRPC, WALLET/MOMO only)
//  5. Persist order + items + history + outbox in one DB tx
//
// Compensation on any failure (in reverse order):
// Payment.Refund → Promotion.ReleaseUsage
func (uc *placeOrderUsecase) placeInline(ctx context.Context, req PlaceOrderRequest, d *orderDraft) (*PlaceOrderResult, error) {
	// ── Step 3: Apply promotions ──────────────────────────────────────────────
	orderID := outbox.NewEventID() // pre-generate so saga steps share it
	var discount int64

	if len(req.VoucherCodes) > 0 && uc.promoClient != nil {
		promoResult, promoErr := uc.promoClient.ApplyPromotion(
			ctx, orderID, req.StoreID, req.CustomerID,
			req.VoucherCodes, d.itemsTotal, d.totalQty,
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

	grandTotal := d.itemsTotal + d.shipFee - discount
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
			uc.compensate(ctx, orderID, grandTotal, false, len(req.VoucherCodes) > 0)
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
			uc.compensate(ctx, orderID, grandTotal, req.PaymentMethod != entity.MethodCOD, true)
			return nil, ErrPromotionInvalid
		}
	}

	return uc.persistPlacedOrder(ctx, req, d, orderID, discount, grandTotal, paymentStatus, payURL)
}

// persistPlacedOrder assigns the order code and writes order + items + status
// history + the frozen order.placed event in one transaction. Both saga
// engines share it; a persistence failure schedules full compensation.
func (uc *placeOrderUsecase) persistPlacedOrder(
	ctx context.Context, req PlaceOrderRequest, d *orderDraft,
	orderID string, discount, grandTotal int64,
	paymentStatus entity.PaymentStatus, payURL string,
) (*PlaceOrderResult, error) {
	traceID := outbox.TraceIDFromCtx(ctx)
	desiredTimePtr := d.desiredTime
	priceMap := d.priceMap
	itemsTotal := d.itemsTotal
	shipFee := d.shipFee
	orderItems := d.orderItems

	codeDay := time.Now().UTC()
	codeSeq, seqErr := uc.orderRepo.NextDailyCodeSeq(ctx, codeDay)
	if seqErr != nil {
		uc.compensate(ctx, orderID, grandTotal, req.PaymentMethod != entity.MethodCOD, len(req.VoucherCodes) > 0)
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
		uc.compensate(ctx, orderID, grandTotal, req.PaymentMethod != entity.MethodCOD, len(req.VoucherCodes) > 0)
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

// compensate rolls back saga side effects after a failed step. Each call is
// attempted immediately; a failed attempt is persisted to
// pending_compensations so the background worker re-drives it until the
// (idempotent, order-id-keyed) call succeeds — a customer can never stay
// charged just because a downstream was unreachable during rollback.
func (uc *placeOrderUsecase) compensate(ctx context.Context, orderID string, amount int64, refund, release bool) {
	if refund && uc.paymentClient != nil {
		if err := uc.paymentClient.Refund(ctx, orderID, amount); err != nil {
			uc.recordCompensation(ctx, orderID, entity.CompensationRefund, amount, err)
		}
	}
	if release && uc.promoClient != nil {
		if err := uc.promoClient.ReleaseUsage(ctx, orderID); err != nil {
			uc.recordCompensation(ctx, orderID, entity.CompensationReleaseUsage, 0, err)
		}
	}
}

// recordCompensation persists a failed rollback intent for the retry worker.
func (uc *placeOrderUsecase) recordCompensation(ctx context.Context, orderID string, action entity.CompensationAction, amount int64, cause error) {
	slog.ErrorContext(ctx, "place order: compensation call failed — persisting for retry",
		"order_id", orderID, "action", action, "err", cause)
	if uc.compRepo == nil {
		slog.ErrorContext(ctx, "place order: compensation repo missing — MANUAL INTERVENTION REQUIRED",
			"order_id", orderID, "action", action)
		return
	}
	msg := cause.Error()
	if err := uc.compRepo.Create(ctx, &entity.PendingCompensation{
		OrderID:   orderID,
		Action:    action,
		Amount:    amount,
		LastError: &msg,
	}); err != nil {
		// Last resort: both the call and the durable record failed. The trace
		// carries order_id + action for manual replay.
		slog.ErrorContext(ctx, "place order: persisting compensation failed — MANUAL INTERVENTION REQUIRED",
			"order_id", orderID, "action", action, "err", err)
	}
}
