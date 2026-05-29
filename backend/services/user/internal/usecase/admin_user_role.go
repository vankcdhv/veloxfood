package usecase

import (
	"context"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
)

// AssignRole delegates to RBACUsecase.AssignRoleToUser. Safeguard: admin cannot
// self-assign (not blocked here — permission system handles it; but we log for audit).
func (uc *adminUserUsecase) AssignRole(
	ctx context.Context,
	adminID, userID, roleID string,
	scopeType entity.ScopeType,
	scopeID *string,
	expiresAt *time.Time,
) error {
	if err := uc.rbacUC.AssignRoleToUser(ctx, adminID, userID, roleID, scopeType, scopeID, expiresAt); err != nil {
		return err
	}
	slog.InfoContext(ctx, "admin assigned role", "admin_id", adminID, "user_id", userID,
		"role_id", roleID, "scope_type", scopeType)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &adminID,
		Action:      "user.role_assigned",
		TargetType:  "user",
		TargetID:    &userID,
		Payload:     map[string]any{"role_id": roleID, "scope_type": string(scopeType)},
	})
	return nil
}

// RemoveRole removes a role from a user via RBACUsecase.
// Safeguard: actor cannot remove their own SUPER_ADMIN assignment.
func (uc *adminUserUsecase) RemoveRole(
	ctx context.Context,
	adminID, userID, roleID string,
	scopeType entity.ScopeType,
	scopeID *string,
) error {
	if adminID == userID && scopeType == entity.ScopeTypeGlobal {
		// Prevent self-lockout when removing a global role.
		return ErrCannotActOnSelf
	}

	if err := uc.rbacUC.RemoveRoleFromUser(ctx, userID, roleID, scopeType, scopeID); err != nil {
		return err
	}
	slog.InfoContext(ctx, "admin removed role", "admin_id", adminID, "user_id", userID,
		"role_id", roleID, "scope_type", scopeType)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &adminID,
		Action:      "user.role_removed",
		TargetType:  "user",
		TargetID:    &userID,
		Payload:     map[string]any{"role_id": roleID, "scope_type": string(scopeType)},
	})
	return nil
}
