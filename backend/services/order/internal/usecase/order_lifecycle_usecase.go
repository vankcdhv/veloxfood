package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/pkg/outbox"
	"project/services/order/internal/entity"
	"project/services/order/internal/infrastructure/grpcclient"
	"project/services/order/internal/repository"

	"gorm.io/gorm"
)

// OrderLifecycleUsecase handles all post-placement state transitions.
type OrderLifecycleUsecase interface {
	GetOrder(ctx context.Context, orderID string) (*entity.Order, error)
	ListCustomerOrders(ctx context.Context, customerID string, page, pageSize int) ([]*entity.Order, int64, error)
	ListStoreOrders(ctx context.Context, storeID string, status entity.OrderStatus, page, pageSize int) ([]*entity.Order, int64, error)

	// AdvanceStatus moves an order to the next status and publishes the matching event.
	// changedBy is the actor's user ID (store owner/staff).
	AdvanceStatus(ctx context.Context, orderID, storeID string, to entity.OrderStatus, changedBy string) error

	// CancelByCustomer cancels a PENDING order and triggers compensation.
	CancelByCustomer(ctx context.Context, orderID, customerID string) error

	// RejectByStore rejects a PENDING order (store cannot fulfil).
	RejectByStore(ctx context.Context, orderID, storeID string) error

	// VerifyPickupPIN marks a READY_PICKUP order as COMPLETED after PIN match.
	VerifyPickupPIN(ctx context.Context, orderID, storeID, pin string) error

	// Reorder creates a new cart pre-filled from a previous order.
	Reorder(ctx context.Context, orderID, customerID string) (*entity.Cart, error)

	// DeliveredAt returns when the order was marked DELIVERED, or nil if not yet.
	DeliveredAt(ctx context.Context, orderID string) (*time.Time, error)
}

type orderLifecycleUsecase struct {
	db            *gorm.DB
	orderRepo     repository.OrderRepository
	cartRepo      repository.CartRepository
	outboxRepo    repository.OutboxRepository
	storeClient   *grpcclient.StoreClient
	promoClient   *grpcclient.PromotionClient
	paymentClient *grpcclient.PaymentClient
	auditLogger   audit.Logger
}

// NewOrderLifecycleUsecase constructs the lifecycle orchestrator.
func NewOrderLifecycleUsecase(
	db *gorm.DB,
	orderRepo repository.OrderRepository,
	cartRepo repository.CartRepository,
	outboxRepo repository.OutboxRepository,
	storeClient *grpcclient.StoreClient,
	promoClient *grpcclient.PromotionClient,
	paymentClient *grpcclient.PaymentClient,
	auditLogger audit.Logger,
) OrderLifecycleUsecase {
	return &orderLifecycleUsecase{
		db:            db,
		orderRepo:     orderRepo,
		cartRepo:      cartRepo,
		outboxRepo:    outboxRepo,
		storeClient:   storeClient,
		promoClient:   promoClient,
		paymentClient: paymentClient,
		auditLogger:   auditLogger,
	}
}

func (uc *orderLifecycleUsecase) GetOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return uc.orderRepo.GetByID(ctx, orderID)
}

func (uc *orderLifecycleUsecase) DeliveredAt(ctx context.Context, orderID string) (*time.Time, error) {
	return uc.orderRepo.StatusChangedAt(ctx, orderID, entity.StatusDelivered)
}

func (uc *orderLifecycleUsecase) ListCustomerOrders(ctx context.Context, customerID string, page, pageSize int) ([]*entity.Order, int64, error) {
	return uc.orderRepo.ListByCustomer(ctx, customerID, page, pageSize)
}

func (uc *orderLifecycleUsecase) ListStoreOrders(ctx context.Context, storeID string, status entity.OrderStatus, page, pageSize int) ([]*entity.Order, int64, error) {
	return uc.orderRepo.ListByStore(ctx, storeID, status, page, pageSize)
}

// AdvanceStatus validates the transition then atomically writes status + event.
func (uc *orderLifecycleUsecase) AdvanceStatus(ctx context.Context, orderID, storeID string, to entity.OrderStatus, changedBy string) error {
	traceID := outbox.TraceIDFromCtx(ctx)
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, err := uc.orderRepo.GetByIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		// The order must belong to the store the caller is authorized for, so an
		// owner cannot advance another store's order via their own store path.
		if order.StoreID != storeID {
			return ErrNotOrderOwner
		}

		if !entity.CanTransition(order.Status, to) {
			return ErrInvalidTransition
		}

		if err := uc.orderRepo.UpdateStatus(ctx, tx, orderID, to, strPtr(changedBy), nil); err != nil {
			return err
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: strPtr(changedBy),
			Action:      "order.status_advanced",
			TargetType:  "order",
			TargetID:    &orderID,
			Payload:     map[string]any{"from": order.Status, "to": to, "store_id": storeID},
		})

		evt := statusEvent(order, to, traceID)
		if evt != nil {
			return uc.outboxRepo.Append(ctx, tx, evt)
		}
		return nil
	})
}

// CancelByCustomer is only allowed when status < CONFIRMED.
// Publishes order.cancelled (frozen), which triggers Payment refund + Store quota restore via consumers.
func (uc *orderLifecycleUsecase) CancelByCustomer(ctx context.Context, orderID, customerID string) error {
	traceID := outbox.TraceIDFromCtx(ctx)
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, err := uc.orderRepo.GetByIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		if order.CustomerID != customerID {
			return ErrNotOrderOwner
		}
		// Customer can only cancel before CONFIRMED.
		switch order.Status {
		case entity.StatusPending:
			// allowed
		default:
			return ErrCancelNotAllowed
		}

		if err := uc.orderRepo.UpdateStatus(ctx, tx, orderID, entity.StatusCancelled, strPtr(customerID), strPtr("cancelled by customer")); err != nil {
			return err
		}
		// payment_status stays PAID here on purpose: the payment service is the
		// source of truth for refunds. It consumes order.cancelled, credits the
		// wallet, then publishes payment.refunded — which flips this order's
		// payment_status. Writing REFUNDED synchronously would claim a refund
		// that may never have happened.
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: strPtr(customerID),
			Action:      "order.cancelled_by_customer",
			TargetType:  "order",
			TargetID:    &orderID,
		})

		evt := cancelledEvent(order, "customer", traceID)
		return uc.outboxRepo.Append(ctx, tx, evt)
	})
}

// RejectByStore rejects a PENDING order before confirmation.
func (uc *orderLifecycleUsecase) RejectByStore(ctx context.Context, orderID, storeID string) error {
	traceID := outbox.TraceIDFromCtx(ctx)
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, err := uc.orderRepo.GetByIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		if order.StoreID != storeID {
			return ErrNotOrderOwner
		}
		if !entity.CanTransition(order.Status, entity.StatusRejected) {
			return ErrInvalidTransition
		}

		if err := uc.orderRepo.UpdateStatus(ctx, tx, orderID, entity.StatusRejected, strPtr(storeID), strPtr("rejected by store")); err != nil {
			return err
		}
		// payment_status flips to REFUNDED via the payment.refunded event once
		// the refund is actually credited (see customer-cancel note above).
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			Action:     "order.rejected_by_store",
			TargetType: "order",
			TargetID:   &orderID,
			Payload:    map[string]any{"store_id": storeID},
		})

		evt := cancelledEvent(order, "store", traceID)
		return uc.outboxRepo.Append(ctx, tx, evt)
	})
}

// VerifyPickupPIN checks PIN and completes the order.
func (uc *orderLifecycleUsecase) VerifyPickupPIN(ctx context.Context, orderID, storeID, pin string) error {
	traceID := outbox.TraceIDFromCtx(ctx)
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, err := uc.orderRepo.GetByIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrOrderNotFound
			}
			return err
		}
		if order.StoreID != storeID {
			return ErrNotOrderOwner
		}
		if order.Status != entity.StatusReadyPickup {
			return ErrInvalidTransition
		}
		if order.PickupPin == nil || *order.PickupPin != pin {
			return ErrPickupPINMismatch
		}

		if err := uc.orderRepo.UpdateStatus(ctx, tx, orderID, entity.StatusCompleted, strPtr(storeID), strPtr("pickup verified")); err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"order_id":    order.ID,
			"code":        order.Code,
			"customer_id": order.CustomerID,
		})
		return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "order",
			AggregateID:   order.ID,
			EventType:     "order.completed",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		})
	})
}

// Reorder pre-fills a new cart from a previous order's items for quick re-purchase.
func (uc *orderLifecycleUsecase) Reorder(ctx context.Context, orderID, customerID string) (*entity.Cart, error) {
	order, err := uc.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if order.CustomerID != customerID {
		return nil, ErrNotOrderOwner
	}

	cart, err := uc.cartRepo.GetOrCreate(ctx, customerID, order.StoreID)
	if err != nil {
		return nil, err
	}
	// Clear existing items first so reorder is a clean replacement.
	if err := uc.cartRepo.ClearCart(ctx, cart.ID); err != nil {
		return nil, err
	}

	for _, oi := range order.Items {
		item := &entity.CartItem{
			CartID:          cart.ID,
			MenuItemID:      oi.MenuItemID,
			NameSnapshot:    oi.NameSnapshot,
			PriceSnapshot:   oi.PriceSnapshot,
			Qty:             oi.Qty,
			OptionsSnapshot: oi.OptionsSnapshot,
		}
		if err := uc.cartRepo.UpsertItem(ctx, item); err != nil {
			slog.WarnContext(ctx, "reorder: upsert item failed", "menu_item_id", oi.MenuItemID, "err", err)
		}
	}

	return uc.cartRepo.GetByCustomerAndStore(ctx, customerID, order.StoreID)
}

// ─── Event builders ────────────────────────────────────────────────────────────

// statusEvent builds the outbox event for a standard lifecycle transition.
// Returns nil for transitions that don't need an external event.
func statusEvent(order *entity.Order, to entity.OrderStatus, traceID string) *entity.OutboxEvent {
	orderID := order.ID
	var eventType string
	var payload []byte

	switch to {
	case entity.StatusConfirmed, entity.StatusPreparing:
		// code + customer_id are required so Notification can fan out to the
		// customer; without them the consumer drops the event silently.
		eventType = "order." + statusEventSuffix(to)
		payload, _ = json.Marshal(map[string]any{
			"order_id":    orderID,
			"code":        order.Code,
			"customer_id": order.CustomerID,
			"status":      string(to),
		})

	case entity.StatusReady:
		// Enriched so the Delivery service can create the delivery record straight
		// from the event (store + destination + fee + desired_time) without a follow-up gRPC call.
		eventType = "order.ready"
		locationID := ""
		if order.LocationID != nil {
			locationID = *order.LocationID
		}
		// location_level defaults to "ROOM" for orders placed before this field existed.
		locationLevel := "ROOM"
		if order.LocationLevel != nil && *order.LocationLevel != "" {
			locationLevel = *order.LocationLevel
		}
		desiredTimeStr := ""
		if order.DesiredTime != nil {
			desiredTimeStr = order.DesiredTime.UTC().Format(time.RFC3339)
		}
		payload, _ = json.Marshal(map[string]any{
			"order_id":       orderID,
			"code":           order.Code,
			"status":         string(entity.StatusReady),
			"store_id":       order.StoreID,
			"location_id":    locationID,
			"location_level": locationLevel,
			"ship_fee":       order.ShipFee,
			"fulfillment":    string(order.Fulfillment),
			"customer_id":    order.CustomerID,
			"desired_time":   desiredTimeStr,
		})

	case entity.StatusReadyPickup:
		eventType = "order.ready_pickup"
		pin := ""
		if order.PickupPin != nil {
			pin = *order.PickupPin
		}
		payload, _ = json.Marshal(map[string]any{"order_id": orderID, "pickup_pin": pin})

	case entity.StatusDelivered:
		eventType = "order.delivered"
		payload, _ = json.Marshal(map[string]any{
			"order_id":       orderID,
			"code":           order.Code,
			"customer_id":    order.CustomerID,
			"store_id":       order.StoreID,
			"amount":         order.GrandTotal,
			"payment_method": string(order.PaymentMethod),
		})

	case entity.StatusCompleted:
		eventType = "order.completed"
		payload, _ = json.Marshal(map[string]any{
			"order_id":    orderID,
			"code":        order.Code,
			"customer_id": order.CustomerID,
		})

	default:
		return nil
	}

	return &entity.OutboxEvent{
		AggregateType: "order",
		AggregateID:   orderID,
		EventType:     eventType,
		Payload:       payload,
		TraceID:       strPtr(traceID),
	}
}

func statusEventSuffix(s entity.OrderStatus) string {
	switch s {
	case entity.StatusConfirmed:
		return "confirmed"
	case entity.StatusPreparing:
		return "preparing"
	case entity.StatusReady:
		return "ready"
	default:
		return string(s)
	}
}

// isPaidOnline reports whether the order's money has already moved (so a cancel
// must trigger a refund): WALLET captures synchronously at placement; MoMo only
// once its IPN marked the order PAID. COD never pre-pays.
func isPaidOnline(order *entity.Order) bool {
	return order.PaymentMethod == entity.MethodWallet ||
		(order.PaymentMethod == entity.MethodMoMo && order.PaymentStatus == entity.PaymentPaid)
}

// cancelledEvent builds the frozen order.cancelled payload (§2bis).
func cancelledEvent(order *entity.Order, cancelledBy string, traceID string) *entity.OutboxEvent {
	wasPaidOnline := isPaidOnline(order)

	items := make([]map[string]any, len(order.Items))
	for i, it := range order.Items {
		items[i] = map[string]any{
			"menu_item_id": it.MenuItemID,
			"qty":          it.Qty,
		}
	}

	payload, _ := json.Marshal(map[string]any{
		"order_id":       order.ID,
		"code":           order.Code,
		"customer_id":    order.CustomerID,
		"store_id":       order.StoreID,
		"cancelled_by":   cancelledBy,
		"was_paid_online": wasPaidOnline,
		"amount":         order.GrandTotal,
		"items":          items,
	})

	return &entity.OutboxEvent{
		AggregateType: "order",
		AggregateID:   order.ID,
		EventType:     "order.cancelled",
		Payload:       payload,
		TraceID:       strPtr(traceID),
	}
}
