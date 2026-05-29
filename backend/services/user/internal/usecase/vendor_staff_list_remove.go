package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"project/pkg/audit"
	"project/services/user/internal/entity"

	"gorm.io/gorm"
)

// ListStaff returns vendor memberships, optionally filtered by status.
func (uc *vendorStaffUsecase) ListStaff(ctx context.Context, vendorID string, status *entity.VendorMembershipStatus) ([]*entity.VendorMembership, error) {
	return uc.membershipRepo.ListByVendor(ctx, vendorID, status)
}

// ListMyVendors returns all vendor memberships for the given user.
func (uc *vendorStaffUsecase) ListMyVendors(ctx context.Context, userID string) ([]*entity.VendorMembership, error) {
	return uc.membershipRepo.ListByUser(ctx, userID)
}

// RemoveStaff sets membership status=left and deletes vendor-scoped role, inside a TX.
// Prevents removing the last owner and an owner removing themselves.
func (uc *vendorStaffUsecase) RemoveStaff(ctx context.Context, removerID, vendorID, targetUserID string) error {
	membership, err := uc.membershipRepo.GetByUserAndVendor(ctx, targetUserID, vendorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMembershipNotFound
		}
		return fmt.Errorf("get membership: %w", err)
	}

	// Owner cannot remove themselves — must transfer ownership first.
	if membership.RoleInVendor == entity.RoleInVendorOwner && targetUserID == removerID {
		return ErrCannotRemoveOwnerAsSelf
	}

	// Prevent removing the last active owner.
	if membership.RoleInVendor == entity.RoleInVendorOwner {
		count, err := uc.membershipRepo.CountOwners(ctx, vendorID)
		if err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	// Resolve role code to find role ID for deletion.
	roleCode, err := mapRoleInVendorToRoleCode(membership.RoleInVendor)
	if err != nil {
		return err
	}
	roleRecord, err := uc.roleRepo.GetByCode(ctx, roleCode)
	if err != nil {
		slog.WarnContext(ctx, "remove staff: role not found — skipping role deletion", "role_code", roleCode, "err", err)
		// Non-fatal: continue with membership status update even if role lookup fails.
		roleRecord = nil
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Set membership status to left.
		if err := tx.WithContext(ctx).
			Model(&entity.VendorMembership{}).
			Where("id = ?", membership.ID).
			Update("status", entity.VendorMembershipLeft).Error; err != nil {
			return fmt.Errorf("update membership status: %w", err)
		}

		// Delete vendor-scoped user_role if role was found.
		if roleRecord != nil {
			if err := tx.WithContext(ctx).
				Where("user_id = ? AND role_id = ? AND scope_type = ? AND scope_id = ?",
					targetUserID, roleRecord.ID, entity.ScopeTypeVendor, vendorID,
				).
				Delete(&entity.UserRole{}).Error; err != nil {
				return fmt.Errorf("delete user role: %w", err)
			}
		}

		// Outbox event.
		payload, _ := json.Marshal(map[string]string{
			"vendor_id": vendorID,
			"user_id":   targetUserID,
			"role":      string(membership.RoleInVendor),
		})
		evt := &entity.OutboxEvent{
			AggregateType: "vendor",
			AggregateID:   vendorID,
			EventType:     "vendor.member.left",
			Payload:       json.RawMessage(payload),
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &removerID,
			Action:      "vendor.staff_removed",
			TargetType:  "vendor",
			TargetID:    &vendorID,
			Payload:     map[string]string{"target_user_id": targetUserID, "role": string(membership.RoleInVendor)},
		})
		return nil
	})

	if txErr != nil {
		return txErr
	}

	uc.rbacUC.InvalidateCacheForUser(ctx, targetUserID)
	slog.InfoContext(ctx, "staff removed from vendor", "vendor_id", vendorID, "target_user_id", targetUserID, "remover_id", removerID)
	return nil
}
