package usecase_test

import (
	"context"
	"testing"
	"time"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"
)

// ---- mock JWTService ----

type mockJWTSvc struct {
	pair authjwt.TokenPair
	err  error
}

func (m *mockJWTSvc) Issue(_ context.Context, _ string) (authjwt.TokenPair, error) {
	return m.pair, m.err
}
func (m *mockJWTSvc) Parse(_ context.Context, _ string) (*authjwt.Claims, error) {
	return nil, nil
}

// ---- userRepo for login tests ----

type loginUserRepo struct {
	user           *entity.User
	findErr        error
	failedAttempts int
	failedErr      error
	lockErr        error
}

func (m *loginUserRepo) Create(_ context.Context, _ *entity.User) error { return nil }
func (m *loginUserRepo) GetByID(_ context.Context, _ string) (*entity.User, error) {
	return m.user, m.findErr
}
func (m *loginUserRepo) GetByEmail(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (m *loginUserRepo) Update(_ context.Context, _ *entity.User) error            { return nil }
func (m *loginUserRepo) Delete(_ context.Context, _ string) error                  { return nil }
func (m *loginUserRepo) List(_ context.Context, _, _ int) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (m *loginUserRepo) FindByEmailOrPhone(_ context.Context, _ string) (*entity.User, error) {
	return m.user, m.findErr
}
func (m *loginUserRepo) UpdateStatus(_ context.Context, _ string, _ entity.UserStatus) error {
	return nil
}
func (m *loginUserRepo) IncrementFailedAttempts(_ context.Context, _ string) (int, error) {
	m.failedAttempts++
	return m.failedAttempts, m.failedErr
}
func (m *loginUserRepo) ResetFailedAttempts(_ context.Context, _ string) error { return nil }
func (m *loginUserRepo) SetLockedUntil(_ context.Context, _ string, _ time.Time) error {
	return m.lockErr
}
func (m *loginUserRepo) UpdatePassword(_ context.Context, _, _ string) error { return nil }
func (m *loginUserRepo) GetByIDs(_ context.Context, _ []string) ([]*entity.User, error) {
	return nil, nil
}

// ---- mock OTPService ----

type mockOTPService struct{}

func (m *mockOTPService) Generate(_ context.Context, _ *string, _ entity.OTPPurpose, _ string) (string, error) {
	return "123456", nil
}
func (m *mockOTPService) Verify(_ context.Context, _ string, _ entity.OTPPurpose, _ string) (*entity.OTPCode, error) {
	return nil, nil
}

// buildLoginUC constructs authUsecase for login tests.
func buildLoginUC(userRepo *loginUserRepo) usecase.AuthLogin {
	jwtSvc := &mockJWTSvc{pair: authjwt.TokenPair{
		JTIAccess:  "jti-access",
		JTIRefresh: "jti-refresh",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	}}
	authStore := &mockAuthStore{}
	return usecase.NewAuthUsecase(
		userRepo,
		nil, // otpRepo — not exercised in login
		nil, // resetRepo — not exercised in login
		jwtSvc,
		authStore,
		nil, // mailer — not exercised
		&mockOTPService{},
		audit.NoopLogger{},
	)
}

// TestLogin_PendingAccount returns ErrAccountPending.
func TestLogin_PendingAccount_ReturnsPendingError(t *testing.T) {
	hash := "anyhash"
	user := &entity.User{
		ID:           "pending-user",
		Status:       entity.UserStatusPending,
		PasswordHash: &hash,
	}
	repo := &loginUserRepo{user: user}
	uc := buildLoginUC(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Identifier: "pending@example.com",
		Password:   "somepass",
	})
	if err != usecase.ErrAccountPending {
		t.Errorf("expected ErrAccountPending, got: %v", err)
	}
}

// TestLogin_UserNotFound returns ErrInvalidCredentials.
func TestLogin_UserNotFound_ReturnsInvalidCredentials(t *testing.T) {
	repo := &loginUserRepo{findErr: usecase.ErrInvalidCredentials}
	uc := buildLoginUC(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Identifier: "nobody@example.com",
		Password:   "pass",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestLogin_EmptyCredentials returns ErrInvalidCredentials immediately.
func TestLogin_EmptyCredentials_ReturnsError(t *testing.T) {
	repo := &loginUserRepo{}
	uc := buildLoginUC(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{})
	if err == nil {
		t.Fatal("expected ErrInvalidCredentials for empty input")
	}
	if err != usecase.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

// TestLogin_AccountLocked returns ErrAccountLocked when LockedUntil is in the future.
func TestLogin_AccountLocked_ReturnsLockedError(t *testing.T) {
	future := time.Now().Add(time.Hour)
	hash := "irrelevant"
	user := &entity.User{
		ID:           "locked-user",
		Status:       entity.UserStatusActive,
		PasswordHash: &hash,
		LockedUntil:  &future,
	}
	repo := &loginUserRepo{user: user}
	uc := buildLoginUC(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Identifier: "locked@example.com",
		Password:   "somepass",
	})
	if err != usecase.ErrAccountLocked {
		t.Errorf("expected ErrAccountLocked, got: %v", err)
	}
}

// TestLogin_FailedAttemptsLockout verifies that maxLoginAttempts failures trigger lockout.
// The mock increments failedAttempts on each IncrementFailedAttempts call.
func TestLogin_FailedAttempts_TriggerLockout(t *testing.T) {
	hash := "badsalt" // not a valid bcrypt hash — bcrypt compare will fail
	user := &entity.User{
		ID:           "user-lockout",
		Status:       entity.UserStatusActive,
		PasswordHash: &hash,
	}
	repo := &loginUserRepo{user: user, failedAttempts: 4} // next call returns 5 → lockout
	uc := buildLoginUC(repo)

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Identifier: "user@example.com",
		Password:   "wrongpass",
	})
	// After 5th fail, should be ErrAccountLocked.
	if err != usecase.ErrAccountLocked {
		t.Errorf("expected ErrAccountLocked on 5th failure, got: %v", err)
	}
}
