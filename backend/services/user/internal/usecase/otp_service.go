package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"project/pkg/mailer"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// OTPService generates and verifies 6-digit OTP codes.
type OTPService interface {
	Generate(ctx context.Context, userID *string, purpose entity.OTPPurpose, destination string) (string, error)
	Verify(ctx context.Context, destination string, purpose entity.OTPPurpose, code string) (*entity.OTPCode, error)
}

type otpService struct {
	repo        repository.OTPRepository
	mailer      mailer.Mailer
	ttl         time.Duration
	maxAttempts int
}

func NewOTPService(
	repo repository.OTPRepository,
	m mailer.Mailer,
	ttl time.Duration,
	maxAttempts int,
) OTPService {
	return &otpService{
		repo:        repo,
		mailer:      m,
		ttl:         ttl,
		maxAttempts: maxAttempts,
	}
}

func (s *otpService) Generate(ctx context.Context, userID *string, purpose entity.OTPPurpose, destination string) (string, error) {
	code := fmt.Sprintf("%06d", rand.Intn(1_000_000)) //nolint:gosec — OTP, not crypto key
	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash otp: %w", err)
	}

	otp := &entity.OTPCode{
		UserID:      userID,
		Purpose:     purpose,
		Destination: destination,
		CodeHash:    string(hash),
		ExpiresAt:   time.Now().Add(s.ttl),
	}
	if err := s.repo.Create(ctx, otp); err != nil {
		return "", fmt.Errorf("store otp: %w", err)
	}

	subject := otpSubject(purpose)
	if err := s.mailer.SendOTP(ctx, destination, subject, code); err != nil {
		// soft fail: OTP persisted, delivery failure is logged, not fatal
		slog.WarnContext(ctx, "otp email send failed", "destination", destination, "err", err)
	}
	return code, nil
}

func (s *otpService) Verify(ctx context.Context, destination string, purpose entity.OTPPurpose, code string) (*entity.OTPCode, error) {
	otp, err := s.repo.FindActive(ctx, destination, purpose)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOTPInvalid
		}
		return nil, err
	}

	if otp.Attempts >= s.maxAttempts {
		return nil, ErrOTPMaxAttempts
	}

	if err := s.repo.IncrementAttempts(ctx, otp.ID); err != nil {
		slog.WarnContext(ctx, "increment otp attempts failed", "id", otp.ID, "err", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)); err != nil {
		return nil, ErrOTPInvalid
	}

	if err := s.repo.MarkUsed(ctx, otp.ID); err != nil {
		return nil, fmt.Errorf("mark otp used: %w", err)
	}
	return otp, nil
}

func otpSubject(purpose entity.OTPPurpose) string {
	switch purpose {
	case entity.OTPPurposeRegisterVerify:
		return "Verify your registration"
	case entity.OTPPurposePasswordReset:
		return "Password reset code"
	case entity.OTPPurposeLogin:
		return "Login verification code"
	case entity.OTPPurposePhoneVerify:
		return "Phone verification code"
	default:
		return "Verification code"
	}
}
