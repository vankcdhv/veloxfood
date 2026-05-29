package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/pkg/auth/store"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// AdminUserUsecase covers admin-initiated user management operations.
type AdminUserUsecase interface {
	Suspend(ctx context.Context, adminID, userID, reason string) error
	Reactivate(ctx context.Context, adminID, userID string) error
	ListUsers(ctx context.Context, filter ListUsersFilter) ([]*entity.User, int64, error)
	AssignRole(ctx context.Context, adminID, userID, roleID string, scopeType entity.ScopeType, scopeID *string, expiresAt *time.Time) error
	RemoveRole(ctx context.Context, adminID, userID, roleID string, scopeType entity.ScopeType, scopeID *string) error
	UpdateStudentByAdmin(ctx context.Context, adminID, userID string, input UpdateStudentAdminInput) error
	UpdateFacultyByAdmin(ctx context.Context, adminID, userID string, input UpdateFacultyAdminInput) error
}

// ListUsersFilter contains optional filter parameters for admin user listing.
type ListUsersFilter struct {
	RoleID   string
	VendorID string
	Status   string
	Search   string
	Page     int
	PageSize int
}

// UpdateStudentAdminInput fields admin is allowed to set on a student profile.
type UpdateStudentAdminInput struct {
	StudentCode   string
	Faculty       *string
	Class         *string
	CohortYear    *int
	DormitoryRoom *string
	Allergies     *json.RawMessage
}

// UpdateFacultyAdminInput fields admin is allowed to set on a faculty profile.
type UpdateFacultyAdminInput struct {
	StaffCode             string
	Department            *string
	Position              *string
	AllowPayrollDeduction *bool
}

type adminUserUsecase struct {
	db          *gorm.DB
	userRepo    repository.UserRepository
	studentRepo repository.StudentProfileRepository
	facultyRepo repository.FacultyProfileRepository
	outboxRepo  repository.OutboxRepository
	rbacUC      RBACUsecase
	authStore   store.AuthStore
	auditLogger audit.Logger
}

// NewAdminUserUsecase constructs AdminUserUsecase.
func NewAdminUserUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	studentRepo repository.StudentProfileRepository,
	facultyRepo repository.FacultyProfileRepository,
	outboxRepo repository.OutboxRepository,
	rbacUC RBACUsecase,
	authStore store.AuthStore,
	auditLogger audit.Logger,
) AdminUserUsecase {
	return &adminUserUsecase{
		db:          db,
		userRepo:    userRepo,
		studentRepo: studentRepo,
		facultyRepo: facultyRepo,
		outboxRepo:  outboxRepo,
		rbacUC:      rbacUC,
		authStore:   authStore,
		auditLogger: auditLogger,
	}
}

// Suspend sets user status to suspended inside a TX, appends outbox event,
// then clears permission cache and revokes active tokens (post-commit, fail-safe).
func (uc *adminUserUsecase) Suspend(ctx context.Context, adminID, userID, reason string) error {
	if adminID == userID {
		return ErrCannotActOnSelf
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if user.Status == entity.UserStatusSuspended {
		return ErrUserAlreadyInStatus
	}

	payload, _ := json.Marshal(map[string]string{"user_id": userID, "reason": reason})
	evt := &entity.OutboxEvent{
		AggregateType: "user",
		AggregateID:   userID,
		EventType:     "user.suspended",
		Payload:       json.RawMessage(payload),
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.User{}).
			Where("id = ?", userID).
			Updates(map[string]any{"status": entity.UserStatusSuspended}).Error; err != nil {
			return fmt.Errorf("update user status: %w", err)
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &adminID,
			Action:      "user.suspended",
			TargetType:  "user",
			TargetID:    &userID,
			Payload:     map[string]string{"reason": reason},
		})
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "suspend: transaction failed", "user_id", userID, "err", txErr)
		return txErr
	}

	// Post-commit: clear cache + revoke tokens. Log errors but do not rollback.
	uc.rbacUC.InvalidateCacheForUser(ctx, userID)
	if err := uc.authStore.RevokeAll(ctx, userID); err != nil {
		slog.ErrorContext(ctx, "suspend: RevokeAll failed — tokens may remain live until TTL",
			"user_id", userID, "err", err)
	}

	slog.InfoContext(ctx, "user suspended", "admin_id", adminID, "user_id", userID, "reason", reason)
	return nil
}

// Reactivate sets user status back to active inside a TX, appends outbox event.
// Does NOT call RevokeAll — user must re-login (tokens were revoked at suspend time).
func (uc *adminUserUsecase) Reactivate(ctx context.Context, adminID, userID string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if user.Status == entity.UserStatusActive {
		return ErrUserAlreadyInStatus
	}

	payload, _ := json.Marshal(map[string]string{"user_id": userID})
	evt := &entity.OutboxEvent{
		AggregateType: "user",
		AggregateID:   userID,
		EventType:     "user.activated",
		Payload:       json.RawMessage(payload),
	}

	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.User{}).
			Where("id = ?", userID).
			Updates(map[string]any{"status": entity.UserStatusActive}).Error; err != nil {
			return fmt.Errorf("update user status: %w", err)
		}
		if err := uc.outboxRepo.Append(ctx, tx, evt); err != nil {
			return fmt.Errorf("append outbox event: %w", err)
		}
		_ = uc.auditLogger.RecordTx(ctx, tx, audit.Entry{
			ActorUserID: &adminID,
			Action:      "user.activated",
			TargetType:  "user",
			TargetID:    &userID,
		})
		return nil
	})
	if txErr != nil {
		slog.ErrorContext(ctx, "reactivate: transaction failed", "user_id", userID, "err", txErr)
		return txErr
	}

	slog.InfoContext(ctx, "user reactivated", "admin_id", adminID, "user_id", userID)
	return nil
}
