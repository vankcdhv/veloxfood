package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	authjwt "project/pkg/auth/jwt"
	"project/services/user/internal/entity"
)

// AuthRefresh handles token rotation.
type AuthRefresh interface {
	Refresh(ctx context.Context, refreshToken string) (authjwt.TokenPair, error)
}

func (uc *authUsecase) Refresh(ctx context.Context, refreshToken string) (authjwt.TokenPair, error) {
	claims, err := uc.jwtSvc.Parse(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, authjwt.ErrTokenExpired) {
			return authjwt.TokenPair{}, ErrTokenInvalid
		}
		return authjwt.TokenPair{}, ErrTokenInvalid
	}

	jtiOld := claims.RegisteredClaims.ID
	userID := claims.UserID

	exists, err := uc.store.Exists(ctx, jtiOld)
	if err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("check jti: %w", err)
	}
	if !exists {
		// Replay attack — revoke all sessions for the user
		slog.WarnContext(ctx, "refresh token replay detected, revoking all sessions", "uid", userID)
		if revokeErr := uc.store.RevokeAll(ctx, userID); revokeErr != nil {
			slog.ErrorContext(ctx, "revoke all sessions failed", "uid", userID, "err", revokeErr)
		}
		return authjwt.TokenPair{}, ErrRefreshLeaked
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return authjwt.TokenPair{}, ErrTokenInvalid
	}
	if user.Status != entity.UserStatusActive {
		return authjwt.TokenPair{}, ErrInvalidCredentials
	}

	newPair, err := uc.jwtSvc.Issue(ctx, userID)
	if err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("issue token: %w", err)
	}

	if err := uc.store.RotatePair(ctx,
		jtiOld, "", // old access JTI unknown from refresh token — only revoke refresh
		newPair.JTIAccess, newPair.JTIRefresh, userID,
		newPair.AccessTTL, newPair.RefreshTTL,
	); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("rotate pair: %w", err)
	}

	slog.InfoContext(ctx, "token pair rotated", "uid", userID)
	return newPair, nil
}
