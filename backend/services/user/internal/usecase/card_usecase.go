package usecase

import (
	"context"
	"errors"
	"log/slog"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// CardUsecase handles card identifier lifecycle (admin operations in Phase 1).
type CardUsecase interface {
	BindCard(ctx context.Context, adminID, userID string, kind entity.CardKind, identifier string) (*entity.CardIdentifier, error)
	RevokeCard(ctx context.Context, adminID, cardID string) error
	ListCardsByUser(ctx context.Context, userID string, includeRevoked bool) ([]*entity.CardIdentifier, error)
}

type cardUsecase struct {
	cardRepo    repository.CardRepository
	userRepo    repository.UserRepository
	auditLogger audit.Logger
}

// NewCardUsecase constructs CardUsecase.
func NewCardUsecase(cardRepo repository.CardRepository, userRepo repository.UserRepository, auditLogger audit.Logger) CardUsecase {
	return &cardUsecase{cardRepo: cardRepo, userRepo: userRepo, auditLogger: auditLogger}
}

// BindCard issues a new card identifier for userID.
// Returns ErrCardConflict if an active card with same kind+identifier already exists.
func (uc *cardUsecase) BindCard(ctx context.Context, adminID, userID string, kind entity.CardKind, identifier string) (*entity.CardIdentifier, error) {
	// Verify target user exists.
	if _, err := uc.userRepo.GetByID(ctx, userID); err != nil {
		return nil, ErrUserNotFound
	}

	dup, err := uc.cardRepo.CheckActiveDuplicate(ctx, kind, identifier)
	if err != nil {
		slog.ErrorContext(ctx, "bind card: duplicate check failed", "err", err)
		return nil, ErrCardCreateFailed
	}
	if dup {
		return nil, ErrCardConflict
	}

	card := &entity.CardIdentifier{
		UserID:     userID,
		Kind:       kind,
		Identifier: identifier,
	}
	if err := uc.cardRepo.Create(ctx, card); err != nil {
		slog.ErrorContext(ctx, "bind card: create failed", "user_id", userID, "err", err)
		return nil, ErrCardCreateFailed
	}
	slog.InfoContext(ctx, "card bound", "card_id", card.ID, "user_id", userID, "kind", kind, "admin_id", adminID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &adminID,
		Action:      "card.bound",
		TargetType:  "card",
		TargetID:    &card.ID,
		Payload:     map[string]string{"user_id": userID, "kind": string(kind)},
	})
	return card, nil
}

// RevokeCard sets revoked_at on the given card.
func (uc *cardUsecase) RevokeCard(ctx context.Context, adminID, cardID string) error {
	if err := uc.cardRepo.Revoke(ctx, cardID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCardNotFound
		}
		slog.ErrorContext(ctx, "revoke card: failed", "card_id", cardID, "err", err)
		return ErrCardCreateFailed
	}
	slog.InfoContext(ctx, "card revoked", "card_id", cardID, "admin_id", adminID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &adminID,
		Action:      "card.revoked",
		TargetType:  "card",
		TargetID:    &cardID,
	})
	return nil
}

// ListCardsByUser returns cards for a user.
func (uc *cardUsecase) ListCardsByUser(ctx context.Context, userID string, includeRevoked bool) ([]*entity.CardIdentifier, error) {
	cards, err := uc.cardRepo.ListByUser(ctx, userID, includeRevoked)
	if err != nil {
		slog.ErrorContext(ctx, "list cards: query failed", "user_id", userID, "err", err)
		return nil, ErrCardCreateFailed
	}
	return cards, nil
}
