package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"project/pkg/outbox"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/infrastructure/grpcclient"
	"project/services/delivery/internal/repository"

	"gorm.io/gorm"
)

// AvailableDeliveryItem is the response shape for the available-deliveries list.
type AvailableDeliveryItem struct {
	OrderID      string `json:"order_id"`
	OrderCode    string `json:"order_code"`
	StoreID      string `json:"store_id"`
	LocationID   string `json:"location_id"`
	RoomPath     string `json:"room_path"`
	ShipFee      int64  `json:"ship_fee"`
	CreatedAt    string `json:"created_at"`
}

// ClaimResult is returned after a successful claim.
type ClaimResult struct {
	BatchID string `json:"batch_id"`
}

// MyDeliveriesResult is the response for the shipper's history + earnings.
type MyDeliveriesResult struct {
	Deliveries []*entity.Delivery `json:"deliveries"`
	Earnings   int64              `json:"earnings"`
}

// DeliveryUsecase handles shipper-facing delivery operations.
type DeliveryUsecase interface {
	ListAvailable(ctx context.Context) ([]*AvailableDeliveryItem, error)
	ClaimDelivery(ctx context.Context, orderID, shipperID string) (*ClaimResult, error)
	UpdateStatus(ctx context.Context, orderID, shipperID string, status entity.DeliveryStatus) error
	MyDeliveries(ctx context.Context, shipperID string) (*MyDeliveriesResult, error)
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
			OrderID:    d.OrderID,
			OrderCode:  d.OrderCode,
			StoreID:    d.StoreID,
			LocationID: d.LocationID,
			RoomPath:   display,
			ShipFee:    d.ShipFee,
			CreatedAt:  d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
	for _, d := range rows {
		if d.Status == entity.DeliveryDelivered {
			earnings += d.ShipFee
		}
	}

	slog.DebugContext(ctx, "delivery: my deliveries loaded",
		"shipper_id", shipperID, "count", len(rows), "earnings", earnings)

	return &MyDeliveriesResult{Deliveries: rows, Earnings: earnings}, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
