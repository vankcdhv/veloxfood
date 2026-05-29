package usecase_test

import (
	"context"
	"testing"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
	"project/services/user/internal/usecase"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---- mock PasswordResetRepository ----

type mockPasswordResetRepo struct {
	record *entity.PasswordResetToken
	err    error
}

func (m *mockPasswordResetRepo) Create(_ context.Context, _ *entity.PasswordResetToken) error {
	return nil
}
func (m *mockPasswordResetRepo) FindByTokenHash(_ context.Context, _ string) (*entity.PasswordResetToken, error) {
	return m.record, m.err
}
func (m *mockPasswordResetRepo) MarkUsed(_ context.Context, _ string) error { return nil }

// ---- mock OTPRepository (minimal) ----

type mockOTPCodeRepo struct{}

func (m *mockOTPCodeRepo) Create(_ context.Context, _ *entity.OTPCode) error { return nil }
func (m *mockOTPCodeRepo) FindActive(_ context.Context, _ string, _ entity.OTPPurpose) (*entity.OTPCode, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockOTPCodeRepo) IncrementAttempts(_ context.Context, _ string) error { return nil }
func (m *mockOTPCodeRepo) MarkUsed(_ context.Context, _ string) error          { return nil }
func (m *mockOTPCodeRepo) CleanExpired(_ context.Context) error                { return nil }
func (m *mockOTPCodeRepo) DeleteExpired(_ context.Context) error               { return nil }

var _ repository.OTPRepository = (*mockOTPCodeRepo)(nil)
var _ repository.PasswordResetRepository = (*mockPasswordResetRepo)(nil)

// buildPasswordUC builds an authUsecase for password tests.
func buildPasswordUC(userRepo *loginUserRepo) usecase.AuthPassword {
	jwtSvc := &mockJWTSvc{pair: authjwt.TokenPair{}}
	return usecase.NewAuthUsecase(
		userRepo,
		&mockOTPCodeRepo{},
		&mockPasswordResetRepo{},
		jwtSvc,
		&mockAuthStore{},
		nil,
		&mockOTPService{},
		audit.NoopLogger{},
	)
}

// TestChangePassword_WeakPassword_ReturnsError validates min-length check.
func TestChangePassword_WeakPassword_ReturnsError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("OldPass123!"), bcrypt.MinCost)
	hashStr := string(hash)
	user := &entity.User{
		ID:           "u1",
		Status:       entity.UserStatusActive,
		PasswordHash: &hashStr,
	}
	repo := &loginUserRepo{user: user}
	uc := buildPasswordUC(repo)

	err := uc.ChangePassword(context.Background(), "u1", "OldPass123!", "short")
	if err != usecase.ErrWeakPassword {
		t.Errorf("expected ErrWeakPassword, got: %v", err)
	}
}

// TestChangePassword_WrongOldPassword_ReturnsError validates bcrypt check.
func TestChangePassword_WrongOldPassword_ReturnsError(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectOld!"), bcrypt.MinCost)
	hashStr := string(hash)
	user := &entity.User{
		ID:           "u2",
		Status:       entity.UserStatusActive,
		PasswordHash: &hashStr,
	}
	repo := &loginUserRepo{user: user}
	uc := buildPasswordUC(repo)

	err := uc.ChangePassword(context.Background(), "u2", "WrongOld!", "NewPassword123!")
	if err != usecase.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

// TestForgotPassword_UnknownIdentifier_SilentlyIgnores verifies no error for unknown email.
func TestForgotPassword_UnknownIdentifier_SilentlyIgnores(t *testing.T) {
	repo := &loginUserRepo{findErr: gorm.ErrRecordNotFound}
	uc := buildPasswordUC(repo)

	err := uc.ForgotPassword(context.Background(), "unknown@example.com")
	if err != nil {
		t.Errorf("expected nil (silent ignore), got: %v", err)
	}
}

// TestForgotPassword_EmptyIdentifier_ReturnsError.
func TestForgotPassword_EmptyIdentifier_ReturnsError(t *testing.T) {
	repo := &loginUserRepo{}
	uc := buildPasswordUC(repo)

	err := uc.ForgotPassword(context.Background(), "")
	if err != usecase.ErrIdentifierRequired {
		t.Errorf("expected ErrIdentifierRequired, got: %v", err)
	}
}

// TestResetPassword_InvalidToken_ReturnsError.
func TestResetPassword_InvalidToken_ReturnsError(t *testing.T) {
	resetRepo := &mockPasswordResetRepo{err: gorm.ErrRecordNotFound}
	jwtSvc := &mockJWTSvc{}
	uc := usecase.NewAuthUsecase(
		&loginUserRepo{},
		&mockOTPCodeRepo{},
		resetRepo,
		jwtSvc,
		&mockAuthStore{},
		nil,
		&mockOTPService{},
		audit.NoopLogger{},
	)

	err := uc.(usecase.AuthPassword).ResetPassword(context.Background(), "badtoken", "NewPassword123!")
	if err != usecase.ErrResetTokenInvalid {
		t.Errorf("expected ErrResetTokenInvalid, got: %v", err)
	}
}

// TestResetPassword_EmptyToken_ReturnsError.
func TestResetPassword_EmptyToken_ReturnsError(t *testing.T) {
	resetRepo := &mockPasswordResetRepo{}
	jwtSvc := &mockJWTSvc{}
	uc := usecase.NewAuthUsecase(
		&loginUserRepo{},
		&mockOTPCodeRepo{},
		resetRepo,
		jwtSvc,
		&mockAuthStore{},
		nil,
		&mockOTPService{},
		audit.NoopLogger{},
	)

	err := uc.(usecase.AuthPassword).ResetPassword(context.Background(), "", "NewPassword123!")
	if err != usecase.ErrResetTokenInvalid {
		t.Errorf("expected ErrResetTokenInvalid, got: %v", err)
	}
}
