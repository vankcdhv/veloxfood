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

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VendorOnboardInput contains the vendor registration details submitted by an owner.
type VendorOnboardInput struct {
	VendorName   string `json:"vendor_name"`
	BusinessType string `json:"business_type"`
	Address      string `json:"address"`
	Phone        string `json:"phone"`
}

// VendorOnboardOutput is returned after a successful onboard request.
type VendorOnboardOutput struct {
	VendorID string `json:"vendor_id"`
	Status   string `json:"status"`
}

// VendorOnboardUsecase handles vendor owner self-onboarding.
type VendorOnboardUsecase interface {
	Onboard(ctx context.Context, ownerUserID string, input VendorOnboardInput) (*VendorOnboardOutput, error)
}

type vendorOnboardUsecase struct {
	db             *gorm.DB
	membershipRepo repository.VendorMembershipRepository
	roleRepo       repository.RoleRepository
	outboxRepo     repository.OutboxRepository
	rbacUC         RBACUsecase
	auditLogger    audit.Logger
}

// NewVendorOnboardUsecase constructs VendorOnboardUsecase.
func NewVendorOnboardUsecase(
	db *gorm.DB,
	membershipRepo repository.VendorMembershipRepository,
	roleRepo repository.RoleRepository,
	outboxRepo repository.OutboxRepository,
	rbacUC RBACUsecase,
	auditLogger audit.Logger,
) VendorOnboardUsecase {
	return &vendorOnboardUsecase{
		db:             db,
		membershipRepo: membershipRepo,
		roleRepo:       roleRepo,
		outboxRepo:     outboxRepo,
		rbacUC:         rbacUC,
		auditLogger:    auditLogger,
	}
}

// Onboard creates a vendor_membership OWNER entry and an outbox event inside a single TX.
// vendor_id is generated here as a placeholder — Vendor Service Phase 2 will consume the event.
func (uc *vendorOnboardUsecase) Onboard(ctx context.Context, ownerUserID string, input VendorOnboardInput) (*VendorOnboardOutput, error) {
	if input.VendorName == "" {
		return nil, ErrVendorNameRequired
	}

	vendorOwnerRole, err := uc.roleRepo.GetByCode(ctx, entity.RoleCodeVendorOwner)
	if err != nil {
		slog.ErrorContext(ctx, "onboard: VENDOR_OWNER role not found", "err", err)
		return nil, ErrVendorRoleNotConfigured
	}

	vendorID := uuid.New().String()
	now := time.Now()

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create vendor membership with status=invited (pending admin approval).
		membership := &entity.VendorMembership{
			UserID:       ownerUserID,
			VendorID:     vendorID,
			RoleInVendor: entity.RoleInVendorOwner,
			Status:       entity.VendorMembershipInvited,
			InvitedBy:    &ownerUserID,
			InvitedAt:    now,
		}
		if err := tx.WithContext(ctx).Create(membership).Error; err != nil {
			return fmt.Errorf("create vendor membership: %w", err)
		}

		// Assign VENDOR_OWNER role scoped to this vendor.
		userRole := &entity.UserRole{
			UserID:    ownerUserID,
			RoleID:    vendorOwnerRole.ID,
			ScopeType: entity.ScopeTypeVendor,
			ScopeID:   &vendorID,
			GrantedBy: &ownerUserID,
		}
		if err := tx.WithContext(ctx).Create(userRole).Error; err != nil {
			return fmt.Errorf("assign vendor owner role: %w", err)
		}

		// Append outbox event for Vendor Service Phase 2 to consume.
		payload, _ := json.Marshal(map[string]string{
			"vendor_id":     vendorID,
			"vendor_name":   input.VendorName,
			"business_type": input.BusinessType,
			"address":       input.Address,
			"phone":         input.Phone,
			"owner_user_id": ownerUserID,
		})
		evt := &entity.OutboxEvent{
			AggregateType: "vendor",
			AggregateID:   vendorID,
			EventType:     "vendor.requested",
			Payload:       json.RawMessage(payload),
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		return nil
	})

	if txErr != nil {
		slog.ErrorContext(ctx, "onboard: transaction failed", "owner_user_id", ownerUserID, "err", txErr)
		return nil, txErr
	}

	uc.rbacUC.InvalidateCacheForUser(ctx, ownerUserID)
	slog.InfoContext(ctx, "vendor onboarded", "vendor_id", vendorID, "owner_user_id", ownerUserID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &ownerUserID,
		Action:      "vendor.onboarded",
		TargetType:  "vendor",
		TargetID:    &vendorID,
		Payload:     map[string]string{"vendor_name": input.VendorName, "business_type": input.BusinessType},
	})

	return &VendorOnboardOutput{
		VendorID: vendorID,
		Status:   "pending_admin_approval",
	}, nil
}
