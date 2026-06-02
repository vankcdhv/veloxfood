package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"project/pkg/apperror"
	"project/pkg/outbox"
	"project/services/store/internal/entity"
	"project/services/store/internal/repository"

	"gorm.io/gorm"
)

// hoursChangePayload is the structured JSON payload stored in
// OperatingHoursChangeRequest.Payload when submitted via the vendor console.
type hoursChangePayload struct {
	OperatingHours []hoursChangeEntry `json:"operating_hours"`
	Note           string             `json:"note"`
}

type hoursChangeEntry struct {
	Weekday   int16  `json:"weekday"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// reHHMM matches strings in "HH:MM" format (00:00–23:59).
var reHHMM = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// validateHoursPayload checks the structured payload for out-of-range values.
// Returns a BadRequest apperror if validation fails.
func validateHoursPayload(p *hoursChangePayload) error {
	for _, oh := range p.OperatingHours {
		if oh.Weekday < 0 || oh.Weekday > 6 {
			return apperror.BadRequest(fmt.Sprintf("operating_hours: weekday must be 0–6, got %d", oh.Weekday))
		}
		if !reHHMM.MatchString(oh.OpenTime) {
			return apperror.BadRequest(fmt.Sprintf("operating_hours: open_time %q is not HH:MM", oh.OpenTime))
		}
		if !reHHMM.MatchString(oh.CloseTime) {
			return apperror.BadRequest(fmt.Sprintf("operating_hours: close_time %q is not HH:MM", oh.CloseTime))
		}
	}
	return nil
}

// OpenNowResult carries the open-now status and today's operating window.
type OpenNowResult struct {
	OpenNow        bool
	OpenTimeToday  string // HH:MM; empty when no hours configured for today
	CloseTimeToday string // HH:MM; empty when no hours configured for today
}

// HoursUsecase manages operating hours and the admin approval workflow
// for hours change requests.
type HoursUsecase interface {
	// Operating hours CRUD
	CreateOperatingHours(ctx context.Context, storeID string, weekday int16, openTime, closeTime string) (*entity.OperatingHours, error)
	ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error)
	UpdateOperatingHours(ctx context.Context, id string, weekday int16, openTime, closeTime string) (*entity.OperatingHours, error)
	DeleteOperatingHours(ctx context.Context, id string) error

	// IsOpenNow returns open status + today's operating window for a store at time t.
	// openNow = store.SaleStatus=="OPEN" AND t falls within today's weekday hours.
	// storeStatus is the current SaleStatus string from the Store entity.
	IsOpenNow(ctx context.Context, storeID string, storeStatus string, t time.Time) (OpenNowResult, error)

	// Hours change approval workflow
	SubmitHoursChange(ctx context.Context, storeID string, payload any) (*entity.OperatingHoursChangeRequest, error)
	ApproveHoursChange(ctx context.Context, reqID, adminID string) error
	RejectHoursChange(ctx context.Context, reqID, adminID string) error
	ListChangeRequests(ctx context.Context, storeID, status string) ([]*entity.OperatingHoursChangeRequest, error)
}

type hoursUsecase struct {
	db           *gorm.DB
	shippingRepo repository.ShippingRepository
	outboxRepo   repository.OutboxRepository
}

func NewHoursUsecase(
	db *gorm.DB,
	shippingRepo repository.ShippingRepository,
	outboxRepo repository.OutboxRepository,
) HoursUsecase {
	return &hoursUsecase{
		db:           db,
		shippingRepo: shippingRepo,
		outboxRepo:   outboxRepo,
	}
}

// ─── Operating hours ─────────────────────────────────────────────────────────

func (uc *hoursUsecase) CreateOperatingHours(ctx context.Context, storeID string, weekday int16, openTime, closeTime string) (*entity.OperatingHours, error) {
	oh := &entity.OperatingHours{StoreID: storeID, Weekday: weekday, OpenTime: openTime, CloseTime: closeTime}
	if err := uc.shippingRepo.CreateOperatingHours(ctx, oh); err != nil {
		return nil, fmt.Errorf("create operating hours: %w", err)
	}
	return oh, nil
}

func (uc *hoursUsecase) ListOperatingHours(ctx context.Context, storeID string) ([]*entity.OperatingHours, error) {
	return uc.shippingRepo.ListOperatingHours(ctx, storeID)
}

func (uc *hoursUsecase) UpdateOperatingHours(ctx context.Context, id string, weekday int16, openTime, closeTime string) (*entity.OperatingHours, error) {
	oh, err := uc.shippingRepo.GetOperatingHours(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStoreNotFound
	}
	if err != nil {
		return nil, err
	}
	oh.Weekday = weekday
	oh.OpenTime = openTime
	oh.CloseTime = closeTime
	if err := uc.shippingRepo.UpdateOperatingHours(ctx, oh); err != nil {
		return nil, fmt.Errorf("update operating hours: %w", err)
	}
	return oh, nil
}

func (uc *hoursUsecase) DeleteOperatingHours(ctx context.Context, id string) error {
	return uc.shippingRepo.DeleteOperatingHours(ctx, id)
}

// IsOpenNow returns whether the store is accepting orders at time t.
// openNow requires both SaleStatus=="OPEN" and t within today's weekday window.
// OpenTimeToday/CloseTimeToday are empty strings when no hours are configured for today.
func (uc *hoursUsecase) IsOpenNow(ctx context.Context, storeID string, storeStatus string, t time.Time) (OpenNowResult, error) {
	allHours, err := uc.shippingRepo.ListOperatingHours(ctx, storeID)
	if err != nil {
		return OpenNowResult{}, fmt.Errorf("list operating hours: %w", err)
	}

	// time.Weekday: Sunday=0 … Saturday=6, matching our entity convention.
	todayWeekday := int16(t.Weekday())
	var todayHours *entity.OperatingHours
	for _, oh := range allHours {
		if oh.Weekday == todayWeekday {
			todayHours = oh
			break
		}
	}

	if todayHours == nil {
		// No hours configured for today — store is effectively closed.
		return OpenNowResult{OpenNow: false}, nil
	}

	res := OpenNowResult{
		OpenTimeToday:  todayHours.OpenTime,
		CloseTimeToday: todayHours.CloseTime,
	}

	if storeStatus != "OPEN" {
		return res, nil
	}

	// Parse open/close as HH:MM and compare against t (same date).
	openMins := parseHHMM(todayHours.OpenTime)
	closeMins := parseHHMM(todayHours.CloseTime)
	nowMins := t.Hour()*60 + t.Minute()

	res.OpenNow = nowMins >= openMins && nowMins <= closeMins
	return res, nil
}

// parseHHMM converts "HH:MM" to total minutes since midnight.
// Returns 0 on malformed input (safe default — compares as midnight).
func parseHHMM(hhmm string) int {
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil {
		return 0
	}
	return h*60 + m
}

// ─── Hours change approval workflow ──────────────────────────────────────────

// SubmitHoursChange creates a pending change request and publishes
// operating_hours.change_requested for admin review.
func (uc *hoursUsecase) SubmitHoursChange(ctx context.Context, storeID string, payload any) (*entity.OperatingHoursChangeRequest, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req := &entity.OperatingHoursChangeRequest{
		StoreID: storeID,
		Payload: string(rawPayload),
		Status:  "pending",
	}

	evtPayload, _ := json.Marshal(map[string]string{"store_id": storeID})
	traceID := outbox.TraceIDFromCtx(ctx)
	evt := &entity.OutboxEvent{
		AggregateType: "store",
		AggregateID:   storeID,
		EventType:     "operating_hours.change_requested",
		Payload:       json.RawMessage(evtPayload),
	}
	if traceID != "" {
		evt.TraceID = &traceID
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(req).Error; err != nil {
			return fmt.Errorf("create change request: %w", err)
		}
		// Include the request ID now that DB has assigned it.
		p, _ := json.Marshal(map[string]string{"store_id": storeID, "request_id": req.ID})
		evt.Payload = json.RawMessage(p)
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return req, nil
}

// ApproveHoursChange marks the request approved, applies the payload to the live
// hours rows, and publishes operating_hours.approved.
func (uc *hoursUsecase) ApproveHoursChange(ctx context.Context, reqID, adminID string) error {
	req, err := uc.shippingRepo.GetChangeRequest(ctx, reqID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrChangeRequestNotFound
	}
	if err != nil {
		return err
	}
	if req.Status != "pending" {
		return ErrInvalidChangeStatus
	}

	evtPayload, _ := json.Marshal(map[string]string{
		"store_id":   req.StoreID,
		"request_id": req.ID,
		"admin_id":   adminID,
	})
	traceID := outbox.TraceIDFromCtx(ctx)
	evt := &entity.OutboxEvent{
		AggregateType: "store",
		AggregateID:   req.StoreID,
		EventType:     "operating_hours.approved",
		Payload:       json.RawMessage(evtPayload),
	}
	if traceID != "" {
		evt.TraceID = &traceID
	}

	// Parse and validate the structured payload before opening the transaction.
	var payload hoursChangePayload
	if err := json.Unmarshal([]byte(req.Payload), &payload); err != nil {
		return apperror.BadRequest(fmt.Sprintf("invalid hours change payload: %v", err))
	}
	if err := validateHoursPayload(&payload); err != nil {
		return err
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Apply operating_hours replacement if the payload carries new entries.
		if len(payload.OperatingHours) > 0 {
			if err := uc.shippingRepo.DeleteOperatingHoursByStore(ctx, tx, req.StoreID); err != nil {
				return fmt.Errorf("delete operating hours: %w", err)
			}
			rows := make([]*entity.OperatingHours, len(payload.OperatingHours))
			for i, oh := range payload.OperatingHours {
				rows[i] = &entity.OperatingHours{
					StoreID:   req.StoreID,
					Weekday:   oh.Weekday,
					OpenTime:  oh.OpenTime,
					CloseTime: oh.CloseTime,
				}
			}
			if err := uc.shippingRepo.BulkCreateOperatingHours(ctx, tx, rows); err != nil {
				return fmt.Errorf("bulk create operating hours: %w", err)
			}
		}

		// Mark the request approved within the same tx.
		if err := tx.Model(&entity.OperatingHoursChangeRequest{}).
			Where("id = ?", reqID).
			Updates(map[string]any{
				"status":      "approved",
				"reviewed_by": adminID,
			}).Error; err != nil {
			return fmt.Errorf("update request status: %w", err)
		}

		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "approve hours change: tx failed", "req_id", reqID, "err", txErr)
		return txErr
	}
	return nil
}

// RejectHoursChange marks the request rejected.
func (uc *hoursUsecase) RejectHoursChange(ctx context.Context, reqID, adminID string) error {
	req, err := uc.shippingRepo.GetChangeRequest(ctx, reqID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrChangeRequestNotFound
	}
	if err != nil {
		return err
	}
	if req.Status != "pending" {
		return ErrInvalidChangeStatus
	}
	return uc.shippingRepo.UpdateChangeRequestStatus(ctx, reqID, "rejected", adminID)
}

func (uc *hoursUsecase) ListChangeRequests(ctx context.Context, storeID, status string) ([]*entity.OperatingHoursChangeRequest, error) {
	return uc.shippingRepo.ListChangeRequests(ctx, storeID, status)
}
