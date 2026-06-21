package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// ShipperListItem enriches a ShipperProfile with the applicant's identity
// fields so the admin list never surfaces raw user_id UUIDs. Fields carry
// explicit snake_case json tags (the web client expects snake_case); embedding
// the entity directly would leak its untagged PascalCase field names.
type ShipperListItem struct {
	UserID             string               `json:"user_id"`
	IDDocumentPhotoURL string               `json:"id_document_photo_url"`
	PortraitPhotoURL   string               `json:"portrait_photo_url"`
	Status             entity.ShipperStatus `json:"status"`
	ApprovedAt         *time.Time           `json:"approved_at"`
	CreatedAt          time.Time            `json:"created_at"`
	FullName           string               `json:"full_name"`
	Email              *string              `json:"email"`
}

// AdminShipperUsecase covers admin approval of shipper applications.
type AdminShipperUsecase interface {
	ListPending(ctx context.Context, status *entity.ShipperStatus, page, pageSize int) ([]*entity.ShipperProfile, int64, error)
	ListPendingEnriched(ctx context.Context, status *entity.ShipperStatus, page, pageSize int) ([]*ShipperListItem, int64, error)
	Approve(ctx context.Context, adminID, shipperUserID string) error
	Reject(ctx context.Context, adminID, shipperUserID, reason string) error
}

type adminShipperUsecase struct {
	db          *gorm.DB
	shipperRepo repository.ShipperProfileRepository
	roleRepo    repository.RoleRepository
	outboxRepo  repository.OutboxRepository
	rbacUC      RBACUsecase
	auditLogger audit.Logger
}

func NewAdminShipperUsecase(
	db *gorm.DB,
	shipperRepo repository.ShipperProfileRepository,
	roleRepo repository.RoleRepository,
	outboxRepo repository.OutboxRepository,
	rbacUC RBACUsecase,
	auditLogger audit.Logger,
) AdminShipperUsecase {
	return &adminShipperUsecase{
		db:          db,
		shipperRepo: shipperRepo,
		roleRepo:    roleRepo,
		outboxRepo:  outboxRepo,
		rbacUC:      rbacUC,
		auditLogger: auditLogger,
	}
}

func (uc *adminShipperUsecase) ListPending(ctx context.Context, status *entity.ShipperStatus, page, pageSize int) ([]*entity.ShipperProfile, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.shipperRepo.List(ctx, status, (page-1)*pageSize, pageSize)
}

// ListPendingEnriched returns shipper profiles joined with the applicant's
// full_name and email so the admin list never renders raw user_id UUIDs.
// A single batch SELECT against the users table avoids N+1 queries.
// Tolerates a missing user record by leaving FullName empty.
func (uc *adminShipperUsecase) ListPendingEnriched(ctx context.Context, status *entity.ShipperStatus, page, pageSize int) ([]*ShipperListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	profiles, total, err := uc.shipperRepo.List(ctx, status, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if len(profiles) == 0 {
		return []*ShipperListItem{}, total, nil
	}

	// Collect user IDs for a single batch lookup.
	ids := make([]string, len(profiles))
	for i, p := range profiles {
		ids[i] = p.UserID
	}

	type userRow struct {
		ID       string
		FullName string
		Email    *string
	}
	var rows []userRow
	if err := uc.db.WithContext(ctx).
		Raw("SELECT id, full_name, email FROM users WHERE id IN ?", ids).
		Scan(&rows).Error; err != nil {
		slog.WarnContext(ctx, "shipper list: user lookup failed, returning profiles without names", "err", err)
		// Degrade gracefully — return profiles with blank names rather than failing.
		rows = nil
	}

	byID := make(map[string]userRow, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}

	items := make([]*ShipperListItem, len(profiles))
	for i, p := range profiles {
		u := byID[p.UserID]
		items[i] = &ShipperListItem{
			UserID:             p.UserID,
			IDDocumentPhotoURL: p.IDDocumentPhotoURL,
			PortraitPhotoURL:   p.PortraitPhotoURL,
			Status:             p.Status,
			ApprovedAt:         p.ApprovedAt,
			CreatedAt:          p.CreatedAt,
			FullName:           u.FullName,
			Email:              u.Email,
		}
	}
	return items, total, nil
}

func (uc *adminShipperUsecase) Approve(ctx context.Context, adminID, shipperUserID string) error {
	if _, err := uc.loadPending(ctx, shipperUserID); err != nil {
		return err
	}

	shipperRole, err := uc.roleRepo.GetByCode(ctx, entity.RoleCodeShipper)
	if err != nil {
		slog.ErrorContext(ctx, "approve shipper: SHIPPER role not found", "err", err)
		return ErrShipperRoleNotConfigured
	}

	now := time.Now()
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.ShipperProfile{}).
			Where("user_id = ? AND status = ?", shipperUserID, entity.ShipperStatusPending).
			Updates(map[string]any{
				"status":      entity.ShipperStatusApproved,
				"approved_by": adminID,
				"approved_at": now,
				"updated_at":  now,
			}).Error; err != nil {
			return fmt.Errorf("update shipper status: %w", err)
		}

		// Grant the global SHIPPER role.
		userRole := &entity.UserRole{
			UserID:    shipperUserID,
			RoleID:    shipperRole.ID,
			ScopeType: entity.ScopeTypeGlobal,
			GrantedBy: &adminID,
		}
		if err := tx.WithContext(ctx).Create(userRole).Error; err != nil {
			return fmt.Errorf("assign shipper role: %w", err)
		}

		payload, _ := json.Marshal(map[string]string{"user_id": shipperUserID, "admin_id": adminID})
		evt := &entity.OutboxEvent{
			AggregateType: "shipper",
			AggregateID:   shipperUserID,
			EventType:     "shipper.approved",
			Payload:       json.RawMessage(payload),
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &adminID,
			Action:      "shipper.approved",
			TargetType:  "shipper",
			TargetID:    &shipperUserID,
		})
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "approve shipper: tx failed", "user_id", shipperUserID, "err", txErr)
		return txErr
	}

	uc.rbacUC.InvalidateCacheForUser(ctx, shipperUserID)
	slog.InfoContext(ctx, "shipper approved", "admin_id", adminID, "user_id", shipperUserID)
	return nil
}

func (uc *adminShipperUsecase) Reject(ctx context.Context, adminID, shipperUserID, reason string) error {
	if _, err := uc.loadPending(ctx, shipperUserID); err != nil {
		return err
	}

	now := time.Now()
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.ShipperProfile{}).
			Where("user_id = ? AND status = ?", shipperUserID, entity.ShipperStatusPending).
			Updates(map[string]any{
				"status":     entity.ShipperStatusRejected,
				"updated_at": now,
			}).Error; err != nil {
			return fmt.Errorf("update shipper status: %w", err)
		}
		payload, _ := json.Marshal(map[string]string{"user_id": shipperUserID, "admin_id": adminID, "reason": reason})
		evt := &entity.OutboxEvent{
			AggregateType: "shipper",
			AggregateID:   shipperUserID,
			EventType:     "shipper.rejected",
			Payload:       json.RawMessage(payload),
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &adminID,
			Action:      "shipper.rejected",
			TargetType:  "shipper",
			TargetID:    &shipperUserID,
			Payload:     map[string]string{"reason": reason},
		})
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "reject shipper: tx failed", "user_id", shipperUserID, "err", txErr)
		return txErr
	}

	slog.InfoContext(ctx, "shipper rejected", "admin_id", adminID, "user_id", shipperUserID, "reason", reason)
	return nil
}

// loadPending fetches the profile and ensures it is still pending.
func (uc *adminShipperUsecase) loadPending(ctx context.Context, userID string) (*entity.ShipperProfile, error) {
	profile, err := uc.shipperRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrShipperNotFound
		}
		return nil, fmt.Errorf("get shipper profile: %w", err)
	}
	if profile.Status != entity.ShipperStatusPending {
		return nil, ErrShipperAlreadyProcessed
	}
	return profile, nil
}
