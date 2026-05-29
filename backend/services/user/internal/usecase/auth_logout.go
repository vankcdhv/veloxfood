package usecase

import (
	"context"
	"log/slog"

	"project/pkg/audit"
)

// AuthLogout revokes both tokens for the current session.
type AuthLogout interface {
	Logout(ctx context.Context, accessToken, refreshToken string) error
}

func (uc *authUsecase) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// Best-effort revoke both tokens; partial failure is logged not fatal.
	revokeOne := func(token string) {
		if token == "" {
			return
		}
		claims, err := uc.jwtSvc.Parse(ctx, token)
		if err != nil {
			// expired/invalid token — still safe to ignore for logout
			return
		}
		jti := claims.RegisteredClaims.ID
		userID := claims.UserID
		if err := uc.store.Revoke(ctx, jti, userID); err != nil {
			slog.WarnContext(ctx, "revoke jti failed", "jti", jti, "err", err)
		}
	}

	revokeOne(accessToken)
	revokeOne(refreshToken)

	slog.InfoContext(ctx, "user logged out")
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		Action:     "auth.logout",
		TargetType: "user",
	})
	return nil
}
