package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

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
	AdvanceStatus(ctx context.Context, orderID string, to entity.OrderStatus, changedBy string) error

	// CancelByCustomer cancels a PENDING order and triggers compensation.
	CancelByCustomer(ctx context.Context, orderID, customerID string) error

	// RejectByStore rejects a PENDING order (store cannot fulfil).
	RejectByStore(ctx context.Context, orderID, storeID string) error

	// VerifyPickupPIN marks a READY_PICKUP order as COMPLETED after PIN match.
	VerifyPickupPIN(ctx context.Context, orderID, storeID, pin string) error

	// Reorder creates a new cart pre-filled from a previous order.
	Reorder(ctx context.Context, orderID, customerID string) (*entity.Cart, error)
}

type orderLifecycleUsecase struct {
	db            *gorm.DB
	orderRepo     repository.OrderRepository
	cartRepo      repository.CartRepository
	outboxRepo    repository.OutboxRepository
	storeClient   *grpcclient.StoreClient
	promoClient   *grpcclient.PromotionClient
	paymentClient *grpcclient.PaymentClient
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
) OrderLifecycleUsecase {
	return &orderLifecycleUsecase{
		db:            db,
		orderRepo:     orderRepo,
		cartRepo:      cartRepo,
		outboxRepo:    outboxRepo,
		storeClient:   storeClient,
		promoClient:   promoClient,
		paymentClient: paymentClient,
	}
}

func (uc *orderLifecycleUsecase) GetOrder(ctx context.Context, orderID string) (*entity.Order, error) {
	return uc.orderRepo.GetByID(ctx, orderID)
}

func (uc *orderLifecycleUsecase) ListCustomerOrders(ctx context.Context, customerID string, page, pageSize int) ([]*entity.Order, int64, error) {
	return uc.orderRepo.ListByCustomer(ctx, customerID, page, pageSize)
}

func (uc *orderLifecycleUsecase) ListStoreOrders(ctx context.Context, storeID string, status entity.OrderStatus, page, pageSize int) ([]*entity.Order, int64, error) {
	return uc.orderRepo.ListByStore(ctx, storeID, status, page, pageSize)
}

// AdvanceStatus validates the transition then atomically writes status + event.
func (uc *orderLifecycleUsecase) AdvanceStatus(ctx context.Context, orderID string, to entity.OrderStatus, changedBy string) error {
	traceID := outbox.TraceIDFromCtx(ctx)
	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		order, err := uc.orderRepo.GetByIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrOrderNotFound
			}
			return err
		}

		if !entity.CanTransition(order.Status, to) {
			return ErrInvalidTransition
		}

		if err := uc.orderRepo.UpdateStatus(ctx, tx, orderID, to, strPtr(changedBy), nil); err != nil {
			return err
		}

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

		payload, _ := json.Marshal(map[string]any{"order_id": order.ID})
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
		eventType = "order." + statusEventSuffix(to)
		payload, _ = json.Marshal(map[string]any{"order_id": orderID, "status": string(to)})

	case entity.StatusReady:
		// Enriched so the Delivery service can create the delivery record straight
		// from the event (store + destination + fee) without a follow-up gRPC call.
		eventType = "order.ready"
		locationID := ""
		if order.LocationID != nil {
			locationID = *order.LocationID
		}
		payload, _ = json.Marshal(map[string]any{
			"order_id":    orderID,
			"status":      string(entity.StatusReady),
			"store_id":    order.StoreID,
			"location_id": locationID,
			"ship_fee":    order.ShipFee,
			"fulfillment": string(order.Fulfillment),
			"customer_id": order.CustomerID,
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
			"store_id":       order.StoreID,
			"amount":         order.GrandTotal,
			"payment_method": string(order.PaymentMethod),
		})

	case entity.StatusCompleted:
		eventType = "order.completed"
		payload, _ = json.Marshal(map[string]any{"order_id": orderID})

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

// cancelledEvent builds the frozen order.cancelled payload (§2bis).
func cancelledEvent(order *entity.Order, cancelledBy string, traceID string) *entity.OutboxEvent {
	wasPaidOnline := order.PaymentMethod == entity.MethodWallet ||
		order.PaymentMethod == entity.MethodMoMo && order.PaymentStatus == entity.PaymentPaid

	items := make([]map[string]any, len(order.Items))
	for i, it := range order.Items {
		m := map[string]any{
			"menu_item_id": it.MenuItemID,
			"qty":          it.Qty,
		}
		if it.CutoffID != nil {
			m["cutoff_id"] = *it.CutoffID
		}
		if it.Date != nil {
			m["date"] = it.Date.Format("2006-01-02")
		}
		items[i] = m
	}

	payload, _ := json.Marshal(map[string]any{
		"order_id":       order.ID,
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
