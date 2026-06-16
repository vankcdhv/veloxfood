package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"project/pkg/outbox"
	"project/services/delivery/internal/entity"
	"project/services/delivery/internal/infrastructure/grpcclient"
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

// AdminIncidentView is the admin-facing shape of a delivery incident: order code
// and shipper name instead of raw UUIDs (snake_case to match the delivery DTOs).
type AdminIncidentView struct {
	ID          string    `json:"id"`
	OrderCode   string    `json:"order_code"`
	ShipperName string    `json:"shipper_name"`
	Type        string    `json:"type"`
	Note        string    `json:"note"`
	PhotoURL    *string   `json:"photo_url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// IncidentUsecase handles incident creation, admin listing, and resolution.
type IncidentUsecase interface {
	ReportIncident(ctx context.Context, req ReportIncidentRequest) (*entity.DeliveryIncident, error)
	ListAllIncidents(ctx context.Context, limit, offset int) ([]*AdminIncidentView, int64, error)
	ResolveIncident(ctx context.Context, id string) error
}

type incidentUsecase struct {
	db           *gorm.DB
	deliveryRepo repository.DeliveryRepository
	incidentRepo repository.IncidentRepository
	outboxRepo   repository.OutboxRepository
	userClient   *grpcclient.UserClient
}

// NewIncidentUsecase constructs the incident usecase. userClient may be nil —
// the admin view then shows an empty shipper name rather than failing.
func NewIncidentUsecase(
	db *gorm.DB,
	deliveryRepo repository.DeliveryRepository,
	incidentRepo repository.IncidentRepository,
	outboxRepo repository.OutboxRepository,
	userClient *grpcclient.UserClient,
) IncidentUsecase {
	return &incidentUsecase{
		db:           db,
		deliveryRepo: deliveryRepo,
		incidentRepo: incidentRepo,
		outboxRepo:   outboxRepo,
		userClient:   userClient,
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

// ListAllIncidents returns a page of admin incidents enriched with each order's
// code and the reporting shipper's name. Shipper names are resolved once per
// distinct shipper to avoid an N+1 of gRPC calls.
func (uc *incidentUsecase) ListAllIncidents(ctx context.Context, limit, offset int) ([]*AdminIncidentView, int64, error) {
	rows, total, err := uc.incidentRepo.ListAllWithOrderCode(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	nameByShipper := make(map[string]string)
	for _, r := range rows {
		if _, ok := nameByShipper[r.ShipperID]; !ok {
			nameByShipper[r.ShipperID] = uc.userClient.GetUser(ctx, r.ShipperID).FullName
		}
	}

	views := make([]*AdminIncidentView, len(rows))
	for i, r := range rows {
		views[i] = &AdminIncidentView{
			ID:          r.ID,
			OrderCode:   r.OrderCode,
			ShipperName: nameByShipper[r.ShipperID],
			Type:        r.Type,
			Note:        r.Note,
			PhotoURL:    r.PhotoURL,
			Status:      string(r.Status),
			CreatedAt:   r.CreatedAt,
		}
	}
	return views, total, nil
}

// ResolveIncident marks an incident RESOLVED (admin "đã xử lý" action).
func (uc *incidentUsecase) ResolveIncident(ctx context.Context, id string) error {
	return uc.incidentRepo.UpdateStatus(ctx, id, entity.IncidentResolved)
}
