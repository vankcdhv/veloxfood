package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/services/user/internal/entity"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthLogin handles credential verification and token issuance.
type AuthLogin interface {
	Login(ctx context.Context, in LoginInput) (authjwt.TokenPair, error)
}

// LoginInput carries email-or-phone + password.
type LoginInput struct {
	Identifier string // email OR phone
	Password   string
}

func (uc *authUsecase) Login(ctx context.Context, in LoginInput) (authjwt.TokenPair, error) {
	if in.Identifier == "" || in.Password == "" {
		return authjwt.TokenPair{}, ErrInvalidCredentials
	}

	user, err := uc.userRepo.FindByEmailOrPhone(ctx, in.Identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return authjwt.TokenPair{}, ErrInvalidCredentials
		}
		return authjwt.TokenPair{}, fmt.Errorf("find user: %w", err)
	}

	// check lockout
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return authjwt.TokenPair{}, ErrAccountLocked
	}

	// status check
	switch user.Status {
	case entity.UserStatusPending:
		return authjwt.TokenPair{}, ErrAccountPending
	case entity.UserStatusSuspended, entity.UserStatusDeactivated:
		return authjwt.TokenPair{}, ErrInvalidCredentials
	}

	if user.PasswordHash == nil {
		return authjwt.TokenPair{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(in.Password)); err != nil {
		return authjwt.TokenPair{}, uc.handleLoginFail(ctx, user)
	}

	// success — reset lockout
	if err := uc.userRepo.ResetFailedAttempts(ctx, user.ID); err != nil {
		slog.WarnContext(ctx, "reset failed attempts error", "uid", user.ID, "err", err)
	}

	pair, err := uc.jwtSvc.Issue(ctx, user.ID)
	if err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("issue token: %w", err)
	}

	if err := uc.store.Whitelist(ctx, pair.JTIAccess, user.ID, pair.AccessTTL); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("whitelist access: %w", err)
	}
	if err := uc.store.Whitelist(ctx, pair.JTIRefresh, user.ID, pair.RefreshTTL); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("whitelist refresh: %w", err)
	}

	slog.InfoContext(ctx, "user logged in", "uid", user.ID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &user.ID,
		Action:      "auth.login_success",
		TargetType:  "user",
		TargetID:    &user.ID,
	})
	return pair, nil
}

// handleLoginFail increments failed attempts and locks account on threshold.
func (uc *authUsecase) handleLoginFail(ctx context.Context, user *entity.User) error {
	count, err := uc.userRepo.IncrementFailedAttempts(ctx, user.ID)
	if err != nil {
		slog.WarnContext(ctx, "increment failed attempts error", "uid", user.ID, "err", err)
		_ = uc.auditLogger.Record(ctx, audit.Entry{
			Action:     "auth.login_failed",
			TargetType: "user",
			TargetID:   &user.ID,
		})
		return ErrInvalidCredentials
	}
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		Action:     "auth.login_failed",
		TargetType: "user",
		TargetID:   &user.ID,
		Payload:    map[string]any{"failed_attempts": count},
	})
	if count >= maxLoginAttempts {
		until := time.Now().Add(lockoutDuration)
		if err := uc.userRepo.SetLockedUntil(ctx, user.ID, until); err != nil {
			slog.WarnContext(ctx, "set locked_until error", "uid", user.ID, "err", err)
		}
		_ = uc.auditLogger.Record(ctx, audit.Entry{
			Action:     "auth.account_locked",
			TargetType: "user",
			TargetID:   &user.ID,
			Payload:    map[string]any{"locked_until": until.Format(time.RFC3339)},
		})
		return ErrAccountLocked
	}
	return ErrInvalidCredentials
}
