package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// AdminVendorUsecase covers school-admin actions on vendor onboarding.
type AdminVendorUsecase interface {
	ApproveVendor(ctx context.Context, adminID, vendorID string) error
	RejectVendor(ctx context.Context, adminID, vendorID, reason string) error
}

type adminVendorUsecase struct {
	db             *gorm.DB
	membershipRepo repository.VendorMembershipRepository
	outboxRepo     repository.OutboxRepository
	rbacUC         RBACUsecase
	auditLogger    audit.Logger
}

// NewAdminVendorUsecase constructs AdminVendorUsecase.
func NewAdminVendorUsecase(
	db *gorm.DB,
	membershipRepo repository.VendorMembershipRepository,
	outboxRepo repository.OutboxRepository,
	rbacUC RBACUsecase,
	auditLogger audit.Logger,
) AdminVendorUsecase {
	return &adminVendorUsecase{
		db:             db,
		membershipRepo: membershipRepo,
		outboxRepo:     outboxRepo,
		rbacUC:         rbacUC,
		auditLogger:    auditLogger,
	}
}

// ApproveVendor transitions all invited memberships for vendorID → active.
// Appends vendor.approved outbox event. Idempotent: if already processed returns error.
func (uc *adminVendorUsecase) ApproveVendor(ctx context.Context, adminID, vendorID string) error {
	invited := entity.VendorMembershipInvited
	members, err := uc.membershipRepo.ListByVendor(ctx, vendorID, &invited)
	if err != nil {
		return fmt.Errorf("list vendor memberships: %w", err)
	}
	if len(members) == 0 {
		// Either vendor doesn't exist or already processed.
		return ErrVendorAlreadyProcessed
	}

	now := time.Now()
	payload, _ := json.Marshal(map[string]string{"vendor_id": vendorID, "admin_id": adminID})
	evt := &entity.OutboxEvent{
		AggregateType: "vendor",
		AggregateID:   vendorID,
		EventType:     "vendor.approved",
		Payload:       json.RawMessage(payload),
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.VendorMembership{}).
			Where("vendor_id = ? AND status = ?", vendorID, entity.VendorMembershipInvited).
			Updates(map[string]any{
				"status":    entity.VendorMembershipActive,
				"joined_at": now,
			}).Error; err != nil {
			return fmt.Errorf("update memberships: %w", err)
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &adminID,
			Action:      "vendor.approved",
			TargetType:  "vendor",
			TargetID:    &vendorID,
		})
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "approve vendor: transaction failed", "vendor_id", vendorID, "err", txErr)
		return txErr
	}

	// Invalidate permission cache for all affected users post-commit.
	for _, m := range members {
		uc.rbacUC.InvalidateCacheForUser(ctx, m.UserID)
	}

	slog.InfoContext(ctx, "vendor approved", "admin_id", adminID, "vendor_id", vendorID,
		"members_activated", len(members))
	return nil
}

// RejectVendor transitions all memberships for vendorID → left, removes vendor-scoped
// user_roles (so owners lose their role), and appends vendor.rejected outbox event.
func (uc *adminVendorUsecase) RejectVendor(ctx context.Context, adminID, vendorID, reason string) error {
	// Collect all members before TX so we can invalidate caches after commit.
	allMembers, err := uc.membershipRepo.ListByVendor(ctx, vendorID, nil)
	if err != nil {
		return fmt.Errorf("list vendor memberships: %w", err)
	}
	if len(allMembers) == 0 {
		return ErrVendorAlreadyProcessed
	}

	payload, _ := json.Marshal(map[string]string{
		"vendor_id": vendorID,
		"admin_id":  adminID,
		"reason":    reason,
	})
	evt := &entity.OutboxEvent{
		AggregateType: "vendor",
		AggregateID:   vendorID,
		EventType:     "vendor.rejected",
		Payload:       json.RawMessage(payload),
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Mark all memberships as left.
		if err := tx.Model(&entity.VendorMembership{}).
			Where("vendor_id = ?", vendorID).
			Update("status", entity.VendorMembershipLeft).Error; err != nil {
			return fmt.Errorf("update memberships: %w", err)
		}

		// Remove all vendor-scoped user_roles for this vendor — cleanup RBAC.
		if err := tx.
			Where("scope_type = ? AND scope_id = ?", entity.ScopeTypeVendor, vendorID).
			Delete(&entity.UserRole{}).Error; err != nil {
			return fmt.Errorf("delete vendor user roles: %w", err)
		}

		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &adminID,
			Action:      "vendor.rejected",
			TargetType:  "vendor",
			TargetID:    &vendorID,
			Payload:     map[string]string{"reason": reason},
		})
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "reject vendor: transaction failed", "vendor_id", vendorID, "err", txErr)
		return txErr
	}

	// Invalidate permission cache for all affected users post-commit.
	for _, m := range allMembers {
		uc.rbacUC.InvalidateCacheForUser(ctx, m.UserID)
	}

	slog.InfoContext(ctx, "vendor rejected", "admin_id", adminID, "vendor_id", vendorID, "reason", reason)
	return nil
}
