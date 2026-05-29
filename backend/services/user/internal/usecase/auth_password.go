package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthPassword groups password management operations.
type AuthPassword interface {
	ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	ForgotPassword(ctx context.Context, identifier string) error
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
}

const resetTokenExpiry = 30 * time.Minute

func (uc *authUsecase) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrInvalidCredentials
	}
	if user.PasswordHash == nil {
		return ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := uc.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Force re-login — all active sessions are invalidated
	if err := uc.store.RevokeAll(ctx, userID); err != nil {
		slog.WarnContext(ctx, "revoke all after password change failed", "uid", userID, "err", err)
	}

	slog.InfoContext(ctx, "password changed", "uid", userID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &userID,
		Action:      "auth.password_changed",
		TargetType:  "user",
		TargetID:    &userID,
	})
	return nil
}

// ForgotPassword silently ignores unknown identifiers to prevent enumeration.
func (uc *authUsecase) ForgotPassword(ctx context.Context, identifier string) error {
	if identifier == "" {
		return ErrIdentifierRequired
	}

	user, err := uc.userRepo.FindByEmailOrPhone(ctx, identifier)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			slog.InfoContext(ctx, "forgot password: identifier not found, silent", "identifier", identifier)
			return nil // generic — no info leak
		}
		return fmt.Errorf("find user: %w", err)
	}

	rawToken, err := generateSecureToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}

	tokenHash := sha256Hex(rawToken)
	resetToken := &entity.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(resetTokenExpiry),
	}
	if err := uc.resetRepo.Create(ctx, resetToken); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	// TODO Phase 03: build URL from app base URL config
	link := fmt.Sprintf("/reset-password?token=%s", rawToken)
	if err := uc.mailer.SendPasswordReset(ctx, identifier, link); err != nil {
		slog.WarnContext(ctx, "password reset email send failed", "err", err)
	}

	slog.InfoContext(ctx, "password reset token issued", "uid", user.ID)
	return nil
}

func (uc *authUsecase) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if rawToken == "" {
		return ErrResetTokenInvalid
	}
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}

	tokenHash := sha256Hex(rawToken)
	record, err := uc.resetRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return ErrResetTokenInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := uc.userRepo.UpdatePassword(ctx, record.UserID, string(hash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	if err := uc.resetRepo.MarkUsed(ctx, record.ID); err != nil {
		slog.WarnContext(ctx, "mark reset token used failed", "id", record.ID, "err", err)
	}

	if err := uc.store.RevokeAll(ctx, record.UserID); err != nil {
		slog.WarnContext(ctx, "revoke all after reset failed", "uid", record.UserID, "err", err)
	}

	slog.InfoContext(ctx, "password reset complete", "uid", record.UserID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &record.UserID,
		Action:      "auth.password_reset",
		TargetType:  "user",
		TargetID:    &record.UserID,
	})
	return nil
}

func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
