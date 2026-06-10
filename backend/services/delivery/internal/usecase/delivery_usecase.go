package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"time"

	"project/pkg/outbox"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/infrastructure/grpcclient"
	"project/services/delivery/internal/repository"

	"gorm.io/gorm"
)

// AvailableDeliveryItem is the response shape for the available-deliveries list.
type AvailableDeliveryItem struct {
	OrderID      string  `json:"order_id"`
	OrderCode    string  `json:"order_code"`
	StoreID      string  `json:"store_id"`
	LocationID   string  `json:"location_id"`
	RoomPath     string  `json:"room_path"`
	ShipFee      int64   `json:"ship_fee"`
	CreatedAt    string  `json:"created_at"`
	// DesiredTime is RFC3339 or empty string when the customer chose ASAP.
	DesiredTime  string  `json:"desired_time"`
	// LateByMinutes is 0 for available/in-progress deliveries (not yet DELIVERED).
	LateByMinutes int64  `json:"late_by_minutes"`
}

// MyDeliveryItem is the per-delivery shape in the shipper's history list.
// It mirrors the Delivery entity but flattens timestamps and adds lateness.
type MyDeliveryItem struct {
	ID            string  `json:"id"`
	OrderID       string  `json:"order_id"`
	OrderCode     string  `json:"order_code"`
	StoreID       string  `json:"store_id"`
	LocationID    string  `json:"location_id"`
	LocationLevel string  `json:"location_level"`
	CustomerID    string  `json:"customer_id"`
	ShipFee       int64   `json:"ship_fee"`
	Status        string  `json:"status"`
	// DesiredTime is RFC3339 or empty string when customer chose ASAP.
	DesiredTime   string  `json:"desired_time"`
	// LateByMinutes is max(0, ceil((delivered_at - desired_time) / min)) when
	// status is DELIVERED and both timestamps are present; 0 otherwise.
	LateByMinutes int64   `json:"late_by_minutes"`
	ClaimedAt     string  `json:"claimed_at"`
	DeliveredAt   string  `json:"delivered_at"`
	CreatedAt     string  `json:"created_at"`
}

// ClaimResult is returned after a successful claim.
type ClaimResult struct {
	BatchID string `json:"batch_id"`
}

// MyDeliveriesResult is the response for the shipper's history + earnings.
type MyDeliveriesResult struct {
	Deliveries []*MyDeliveryItem `json:"deliveries"`
	Earnings   int64             `json:"earnings"`
}

// DeliveryUsecase handles shipper-facing delivery operations.
type DeliveryUsecase interface {
	ListAvailable(ctx context.Context) ([]*AvailableDeliveryItem, error)
	ClaimDelivery(ctx context.Context, orderID, shipperID string) (*ClaimResult, error)
	UpdateStatus(ctx context.Context, orderID, shipperID string, status entity.DeliveryStatus) error
	MyDeliveries(ctx context.Context, shipperID string) (*MyDeliveriesResult, error)
	// GetOrderShipper returns the shipper assigned to the caller's order (for
	// rating). Empty when the caller doesn't own the order or none is assigned.
	GetOrderShipper(ctx context.Context, orderID, customerID string) (string, error)
}

type deliveryUsecase struct {
	db             *gorm.DB
	deliveryRepo   repository.DeliveryRepository
	batchRepo      repository.BatchRepository
	outboxRepo     repository.OutboxRepository
	locationClient *grpcclient.LocationClient
}

// NewDeliveryUsecase constructs the delivery usecase.
func NewDeliveryUsecase(
	db *gorm.DB,
	deliveryRepo repository.DeliveryRepository,
	batchRepo repository.BatchRepository,
	outboxRepo repository.OutboxRepository,
	locationClient *grpcclient.LocationClient,
) DeliveryUsecase {
	return &deliveryUsecase{
		db:             db,
		deliveryRepo:   deliveryRepo,
		batchRepo:      batchRepo,
		outboxRepo:     outboxRepo,
		locationClient: locationClient,
	}
}

func (uc *deliveryUsecase) ListAvailable(ctx context.Context) ([]*AvailableDeliveryItem, error) {
	rows, err := uc.deliveryRepo.ListAvailable(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*AvailableDeliveryItem, len(rows))
	for i, d := range rows {
		// Branch by location_level so shippers see the right path granularity:
		// BUILDING → "Toà A", FLOOR → "Toà A / Tầng 2", ROOM → full path.
		var display string
		if uc.locationClient != nil {
			display = uc.locationClient.GetLocationPath(ctx, d.LocationID, d.LocationLevel)
		}
		if display == "" {
			display = d.LocationID // graceful fallback: show UUID
		}
		items[i] = &AvailableDeliveryItem{
			OrderID:       d.OrderID,
			OrderCode:     d.OrderCode,
			StoreID:       d.StoreID,
			LocationID:    d.LocationID,
			RoomPath:      display,
			ShipFee:       d.ShipFee,
			CreatedAt:     d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			DesiredTime:   formatTimePtr(d.DesiredTime),
			LateByMinutes: 0, // AVAILABLE deliveries are not yet delivered
		}
	}
	return items, nil
}

// ClaimDelivery atomically assigns the delivery to the shipper.
// Enforces: delivery must be AVAILABLE with no shipper (atomic UPDATE);
// shipper's OPEN batch for the store must have < MaxSize active deliveries.
func (uc *deliveryUsecase) ClaimDelivery(ctx context.Context, orderID, shipperID string) (*ClaimResult, error) {
	traceID := outbox.TraceIDFromCtx(ctx)

	var result *ClaimResult
	err := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the delivery row first to read store_id for batch lookup.
		delivery, err := uc.deliveryRepo.GetByOrderIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrDeliveryNotFound
			}
			return err
		}
		if delivery.Status != entity.DeliveryAvailable || delivery.ShipperID != nil {
			return ErrAlreadyClaimed
		}

		// ── Batch enforcement ──────────────────────────────────────────────────
		batch, err := uc.batchRepo.FindOpenBatch(ctx, tx, shipperID, delivery.StoreID)
		if err != nil {
			return err
		}

		var batchID string
		if batch != nil {
			// Check active count against max size.
			count, err := uc.deliveryRepo.CountBatchActiveDeliveries(ctx, tx, batch.ID)
			if err != nil {
				return err
			}
			if count >= int64(batch.MaxSize) {
				return ErrBatchFull
			}
			batchID = batch.ID
		} else {
			// Create a new OPEN batch for this (shipper, store).
			newBatch := &entity.DeliveryBatch{
				ShipperID: shipperID,
				StoreID:   delivery.StoreID,
				MaxSize:   5,
				Status:    entity.BatchOpen,
			}
			if err := uc.batchRepo.CreateBatch(ctx, tx, newBatch); err != nil {
				return err
			}
			batchID = newBatch.ID
		}

		// ── Atomic claim: UPDATE WHERE shipper_id IS NULL AND status='AVAILABLE' ──
		affected, err := uc.deliveryRepo.ClaimDelivery(ctx, tx, orderID, shipperID, batchID)
		if err != nil {
			return err
		}
		if affected == 0 {
			// Another shipper claimed between our SELECT FOR UPDATE and here — extremely
			// unlikely but the WHERE guard ensures correctness.
			return ErrAlreadyClaimed
		}

		// ── Publish delivery.claimed via outbox ────────────────────────────────
		payload, _ := json.Marshal(map[string]any{
			"order_id":   orderID,
			"shipper_id": shipperID,
			"batch_id":   batchID,
		})
		if err := uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "delivery",
			AggregateID:   delivery.ID,
			EventType:     "delivery.claimed",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		}); err != nil {
			return err
		}

		result = &ClaimResult{BatchID: batchID}
		return nil
	})
	return result, err
}

// UpdateStatus advances the delivery status (CLAIMED→PICKED_UP→DELIVERING→DELIVERED).
// Only the claiming shipper may update.
func (uc *deliveryUsecase) UpdateStatus(ctx context.Context, orderID, shipperID string, next entity.DeliveryStatus) error {
	traceID := outbox.TraceIDFromCtx(ctx)

	return uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		delivery, err := uc.deliveryRepo.GetByOrderIDForUpdate(ctx, tx, orderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrDeliveryNotFound
			}
			return err
		}

		if delivery.ShipperID == nil || *delivery.ShipperID != shipperID {
			return ErrNotDeliveryOwner
		}

		if !delivery.CanTransitionTo(next) {
			return ErrInvalidTransition
		}

		// DELIVERED needs its own update (sets delivered_at).
		if next == entity.DeliveryDelivered {
			if err := uc.deliveryRepo.MarkDelivered(ctx, tx, orderID); err != nil {
				return err
			}
		} else {
			if err := uc.deliveryRepo.UpdateStatus(ctx, tx, orderID, next, nil, nil); err != nil {
				return err
			}
		}

		// Publish delivery.status_changed for all shipper-driven transitions.
		payload, _ := json.Marshal(map[string]any{
			"order_id":   orderID,
			"status":     string(next),
			"shipper_id": shipperID,
		})
		return uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "delivery",
			AggregateID:   delivery.ID,
			EventType:     "delivery.status_changed",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		})
	})
}

// MyDeliveries returns all deliveries for the shipper plus total earnings (Σ ship_fee where DELIVERED).
func (uc *deliveryUsecase) MyDeliveries(ctx context.Context, shipperID string) (*MyDeliveriesResult, error) {
	rows, err := uc.deliveryRepo.ListByShipper(ctx, shipperID)
	if err != nil {
		return nil, err
	}

	var earnings int64
	items := make([]*MyDeliveryItem, len(rows))
	for i, d := range rows {
		if d.Status == entity.DeliveryDelivered {
			earnings += d.ShipFee
		}
		items[i] = &MyDeliveryItem{
			ID:            d.ID,
			OrderID:       d.OrderID,
			OrderCode:     d.OrderCode,
			StoreID:       d.StoreID,
			LocationID:    d.LocationID,
			LocationLevel: d.LocationLevel,
			CustomerID:    d.CustomerID,
			ShipFee:       d.ShipFee,
			Status:        string(d.Status),
			DesiredTime:   formatTimePtr(d.DesiredTime),
			LateByMinutes: computeLateMinutes(d),
			ClaimedAt:     formatTimePtr(d.ClaimedAt),
			DeliveredAt:   formatTimePtr(d.DeliveredAt),
			CreatedAt:     d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	slog.DebugContext(ctx, "delivery: my deliveries loaded",
		"shipper_id", shipperID, "count", len(rows), "earnings", earnings)

	return &MyDeliveriesResult{Deliveries: items, Earnings: earnings}, nil
}

// GetOrderShipper returns the shipper assigned to the order, but only to the
// customer who owns it (so they can rate the delivery). Returns "" when there
// is no delivery, no shipper yet, or the caller is not the order's customer.
func (uc *deliveryUsecase) GetOrderShipper(ctx context.Context, orderID, customerID string) (string, error) {
	d, err := uc.deliveryRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	if d.CustomerID != customerID || d.ShipperID == nil {
		return "", nil
	}
	return *d.ShipperID, nil
}

// formatTimePtr formats a *time.Time as RFC3339; returns "" when nil.
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}

// computeLateMinutes returns max(0, round((DeliveredAt - DesiredTime) / minute))
// only when status is DELIVERED and both timestamps are present; otherwise 0.
func computeLateMinutes(d *entity.Delivery) int64 {
	if d.Status != entity.DeliveryDelivered || d.DeliveredAt == nil || d.DesiredTime == nil {
		return 0
	}
	diff := d.DeliveredAt.Sub(*d.DesiredTime)
	minutes := int64(math.Round(diff.Minutes()))
	if minutes < 0 {
		return 0
	}
	return minutes
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
