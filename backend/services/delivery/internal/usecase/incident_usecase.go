package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"project/pkg/outbox"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/repository"

	"gorm.io/gorm"
)

// ReportIncidentRequest contains the data for a new incident report.
type ReportIncidentRequest struct {
	OrderID   string
	ShipperID string
	Type      string
	Note      string
	PhotoURL  *string
}

// IncidentUsecase handles incident creation and admin listing.
type IncidentUsecase interface {
	ReportIncident(ctx context.Context, req ReportIncidentRequest) (*entity.DeliveryIncident, error)
	ListAllIncidents(ctx context.Context) ([]*entity.DeliveryIncident, error)
}

type incidentUsecase struct {
	db             *gorm.DB
	deliveryRepo   repository.DeliveryRepository
	incidentRepo   repository.IncidentRepository
	outboxRepo     repository.OutboxRepository
}

// NewIncidentUsecase constructs the incident usecase.
func NewIncidentUsecase(
	db *gorm.DB,
	deliveryRepo repository.DeliveryRepository,
	incidentRepo repository.IncidentRepository,
	outboxRepo repository.OutboxRepository,
) IncidentUsecase {
	return &incidentUsecase{
		db:           db,
		deliveryRepo: deliveryRepo,
		incidentRepo: incidentRepo,
		outboxRepo:   outboxRepo,
	}
}

// ReportIncident creates an incident record and publishes delivery.incident_reported.
// Only the claiming shipper may report an incident on their delivery.
func (uc *incidentUsecase) ReportIncident(ctx context.Context, req ReportIncidentRequest) (*entity.DeliveryIncident, error) {
	traceID := outbox.TraceIDFromCtx(ctx)

	var created *entity.DeliveryIncident
	err := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		delivery, err := uc.deliveryRepo.GetByOrderIDForUpdate(ctx, tx, req.OrderID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrDeliveryNotFound
			}
			return err
		}

		if delivery.ShipperID == nil || *delivery.ShipperID != req.ShipperID {
			return ErrNotDeliveryOwner
		}

		inc := &entity.DeliveryIncident{
			DeliveryID: delivery.ID,
			OrderID:    req.OrderID,
			ShipperID:  req.ShipperID,
			Type:       req.Type,
			Note:       req.Note,
			PhotoURL:   req.PhotoURL,
			Status:     entity.IncidentOpen,
		}

		if err := uc.incidentRepo.Create(ctx, tx, inc); err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]any{
			"order_id":    req.OrderID,
			"delivery_id": delivery.ID,
			"shipper_id":  req.ShipperID,
			"type":        req.Type,
			"note":        req.Note,
		})
		if err := uc.outboxRepo.Append(ctx, tx, &entity.OutboxEvent{
			AggregateType: "delivery",
			AggregateID:   delivery.ID,
			EventType:     "delivery.incident_reported",
			Payload:       payload,
			TraceID:       strPtr(traceID),
		}); err != nil {
			return err
		}

		created = inc
		return nil
	})
	return created, err
}

func (uc *incidentUsecase) ListAllIncidents(ctx context.Context) ([]*entity.DeliveryIncident, error) {
	return uc.incidentRepo.ListAll(ctx)
}
