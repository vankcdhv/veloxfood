package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"project/pkg/audit"
	"project/services/user/internal/entity"

	"gorm.io/gorm"
)

// Accept claims a pending invitation and provisions membership + role inside a TX.
// Implements the atomic accept pattern: only one concurrent goroutine wins the UPDATE.
func (uc *vendorStaffUsecase) Accept(ctx context.Context, userID string, input AcceptInvitationInput) (*AcceptInvitationOutput, error) {
	if input.Token == "" {
		return nil, ErrInvitationNotFound
	}

	hashBytes := sha256.Sum256([]byte(input.Token))
	tokenHash := hex.EncodeToString(hashBytes[:])

	var result AcceptInvitationOutput

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Atomic claim — only one goroutine wins if two arrive concurrently.
		inv, err := uc.invitationRepo.AcceptAtomic(ctx, tokenHash, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvitationInvalidOrExpired
			}
			return fmt.Errorf("accept invitation: %w", err)
		}

		// Resolve system role code for the vendor role.
		roleCode, err := mapRoleInVendorToRoleCode(inv.RoleInVendor)
		if err != nil {
			return err
		}
		roleRecord, err := uc.roleRepo.GetByCode(ctx, roleCode)
		if err != nil {
			slog.ErrorContext(ctx, "accept: role code not found in DB", "role_code", roleCode, "err", err)
			return ErrVendorRoleNotConfigured
		}

		// Upsert active membership.
		membership := &entity.VendorMembership{
			UserID:       userID,
			VendorID:     inv.VendorID,
			RoleInVendor: inv.RoleInVendor,
			Status:       entity.VendorMembershipActive,
			InvitedBy:    nil,
		}
		if err := tx.WithContext(ctx).
			Exec(`INSERT INTO vendor_memberships (user_id, vendor_id, role_in_vendor, status, invited_at)
			      VALUES (?, ?, ?, ?, NOW())
			      ON CONFLICT (user_id, vendor_id) DO UPDATE
			        SET status = 'active', role_in_vendor = EXCLUDED.role_in_vendor, joined_at = NOW()`,
				membership.UserID, membership.VendorID, membership.RoleInVendor, entity.VendorMembershipActive,
			).Error; err != nil {
			return fmt.Errorf("upsert vendor membership: %w", err)
		}

		// Assign role — ON CONFLICT DO NOTHING (idempotent).
		if err := tx.WithContext(ctx).
			Exec(`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
			      VALUES (?, ?, ?, ?)
			      ON CONFLICT DO NOTHING`,
				userID, roleRecord.ID, entity.ScopeTypeVendor, inv.VendorID,
			).Error; err != nil {
			return fmt.Errorf("assign user role: %w", err)
		}

		// Outbox event.
		payload, _ := json.Marshal(map[string]string{
			"vendor_id": inv.VendorID,
			"user_id":   userID,
			"role":      string(inv.RoleInVendor),
		})
		evt := &entity.OutboxEvent{
			AggregateType: "vendor",
			AggregateID:   inv.VendorID,
			EventType:     "vendor.member.joined",
			Payload:       json.RawMessage(payload),
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &userID,
			Action:      "vendor.invite_accepted",
			TargetType:  "vendor",
			TargetID:    &inv.VendorID,
			Payload:     map[string]string{"role": string(inv.RoleInVendor)},
		})

		result = AcceptInvitationOutput{
			VendorID: inv.VendorID,
			Role:     inv.RoleInVendor,
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	uc.rbacUC.InvalidateCacheForUser(ctx, userID)
	slog.InfoContext(ctx, "invitation accepted", "user_id", userID, "vendor_id", result.VendorID)
	return &result, nil
}

// mapRoleInVendorToRoleCode maps RoleInVendor to the system role code string.
// Vendor RBAC has exactly two roles: Owner and Staff.
func mapRoleInVendorToRoleCode(r entity.RoleInVendor) (string, error) {
	switch r {
	case entity.RoleInVendorOwner:
		return entity.RoleCodeVendorOwner, nil
	case entity.RoleInVendorStaff:
		return entity.RoleCodeVendorStaff, nil
	default:
		return "", ErrInvalidInviteRole
	}
}
