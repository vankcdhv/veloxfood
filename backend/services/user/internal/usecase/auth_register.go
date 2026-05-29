package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/services/user/internal/entity"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthRegister groups registration + OTP verification operations.
type AuthRegister interface {
	Register(ctx context.Context, in RegisterInput) (*entity.User, error)
	VerifyRegister(ctx context.Context, in VerifyRegisterInput) (authjwt.TokenPair, error)
}

// RegisterInput is the payload for new user registration.
type RegisterInput struct {
	Email    string
	Phone    string
	Password string
	FullName string
}

// VerifyRegisterInput carries the OTP verification request.
type VerifyRegisterInput struct {
	// Destination is email or phone — whichever was registered.
	Destination string
	Code        string
}

func (uc *authUsecase) Register(ctx context.Context, in RegisterInput) (*entity.User, error) {
	if in.Password == "" {
		return nil, ErrPasswordRequired
	}
	if len(in.Password) < 8 {
		return nil, ErrWeakPassword
	}
	if in.Email == "" && in.Phone == "" {
		return nil, ErrIdentifierRequired
	}

	// Uniqueness check — generic error on collision
	identifier := in.Email
	if identifier == "" {
		identifier = in.Phone
	}
	if existing, err := uc.userRepo.FindByEmailOrPhone(ctx, identifier); err == nil && existing != nil {
		return nil, ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &entity.User{
		FullName:     in.FullName,
		Status:       entity.UserStatusPending,
		PasswordHash: strPtr(string(hash)),
	}
	if in.Email != "" {
		user.Email = strPtr(in.Email)
	}
	if in.Phone != "" {
		user.Phone = strPtr(in.Phone)
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	slog.InfoContext(ctx, "user registered", "id", user.ID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &user.ID,
		Action:      "auth.register",
		TargetType:  "user",
		TargetID:    &user.ID,
	})

	destination := in.Email
	if destination == "" {
		destination = in.Phone
	}
	if _, err := uc.otpSvc.Generate(ctx, &user.ID, entity.OTPPurposeRegisterVerify, destination); err != nil {
		slog.WarnContext(ctx, "otp generate failed after register", "err", err)
	}

	return user, nil
}

func (uc *authUsecase) VerifyRegister(ctx context.Context, in VerifyRegisterInput) (authjwt.TokenPair, error) {
	otp, err := uc.otpSvc.Verify(ctx, in.Destination, entity.OTPPurposeRegisterVerify, in.Code)
	if err != nil {
		return authjwt.TokenPair{}, err
	}

	var userID string
	if otp.UserID != nil {
		userID = *otp.UserID
	} else {
		// Fallback: look up by email/phone
		u, err := uc.userRepo.FindByEmailOrPhone(ctx, in.Destination)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return authjwt.TokenPair{}, ErrInvalidCredentials
			}
			return authjwt.TokenPair{}, err
		}
		userID = u.ID
	}

	if err := uc.userRepo.UpdateStatus(ctx, userID, entity.UserStatusActive); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("activate user: %w", err)
	}

	pair, err := uc.jwtSvc.Issue(ctx, userID)
	if err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("issue token: %w", err)
	}

	if err := uc.store.Whitelist(ctx, pair.JTIAccess, userID, pair.AccessTTL); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("whitelist access: %w", err)
	}
	if err := uc.store.Whitelist(ctx, pair.JTIRefresh, userID, pair.RefreshTTL); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("whitelist refresh: %w", err)
	}

	slog.InfoContext(ctx, "user verified", "id", userID)
	return pair, nil
}

func strPtr(s string) *string { return &s }
