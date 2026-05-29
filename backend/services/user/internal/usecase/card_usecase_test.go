package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"gorm.io/gorm"
)

// ---- mock CardRepository ----

type mockCardRepo struct {
	cards     []*entity.CardIdentifier
	createErr error
	revokeErr error
	dupExists bool
	dupErr    error
}

func (m *mockCardRepo) Create(_ context.Context, c *entity.CardIdentifier) error {
	if m.createErr != nil {
		return m.createErr
	}
	c.ID = "card-uuid-1"
	m.cards = append(m.cards, c)
	return nil
}

func (m *mockCardRepo) GetByID(_ context.Context, id string) (*entity.CardIdentifier, error) {
	for _, c := range m.cards {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockCardRepo) ListByUser(_ context.Context, _ string, _ bool) ([]*entity.CardIdentifier, error) {
	return m.cards, nil
}

func (m *mockCardRepo) Revoke(_ context.Context, _ string) error {
	return m.revokeErr
}

func (m *mockCardRepo) CheckActiveDuplicate(_ context.Context, _ entity.CardKind, _ string) (bool, error) {
	return m.dupExists, m.dupErr
}

// ---- tests ----

func TestBindCard_Success(t *testing.T) {
	cardRepo := &mockCardRepo{}
	userRepo := &mockUserRepo{user: newTestUser()}
	uc := usecase.NewCardUsecase(cardRepo, userRepo, audit.NoopLogger{})

	card, err := uc.BindCard(context.Background(), "admin-1", "user-uuid-1", entity.CardKindRFID, "RFID-ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.ID == "" {
		t.Error("expected card ID to be set")
	}
	if card.UserID != "user-uuid-1" {
		t.Errorf("expected user_id=user-uuid-1, got %s", card.UserID)
	}
}

func TestBindCard_DuplicateActive_Returns409(t *testing.T) {
	cardRepo := &mockCardRepo{dupExists: true}
	userRepo := &mockUserRepo{user: newTestUser()}
	uc := usecase.NewCardUsecase(cardRepo, userRepo, audit.NoopLogger{})

	_, err := uc.BindCard(context.Background(), "admin-1", "user-uuid-1", entity.CardKindRFID, "RFID-ABC")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !errors.Is(err, usecase.ErrCardConflict) {
		t.Errorf("expected ErrCardConflict, got: %v", err)
	}
}

func TestBindCard_UserNotFound(t *testing.T) {
	cardRepo := &mockCardRepo{}
	userRepo := &mockUserRepo{err: errors.New("not found")}
	uc := usecase.NewCardUsecase(cardRepo, userRepo, audit.NoopLogger{})

	_, err := uc.BindCard(context.Background(), "admin-1", "bad-user", entity.CardKindRFID, "RFID-XYZ")
	if err == nil {
		t.Fatal("expected error when user not found")
	}
}

func TestRevokeCard_Success(t *testing.T) {
	now := time.Now()
	cardRepo := &mockCardRepo{
		cards: []*entity.CardIdentifier{
			{ID: "card-1", UserID: "user-uuid-1", Kind: entity.CardKindRFID, Identifier: "A", IssuedAt: now},
		},
	}
	userRepo := &mockUserRepo{user: newTestUser()}
	uc := usecase.NewCardUsecase(cardRepo, userRepo, audit.NoopLogger{})

	if err := uc.RevokeCard(context.Background(), "admin-1", "card-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRevokeCard_NotFound(t *testing.T) {
	cardRepo := &mockCardRepo{revokeErr: gorm.ErrRecordNotFound}
	userRepo := &mockUserRepo{user: newTestUser()}
	uc := usecase.NewCardUsecase(cardRepo, userRepo, audit.NoopLogger{})

	err := uc.RevokeCard(context.Background(), "admin-1", "missing-card")
	if !errors.Is(err, usecase.ErrCardNotFound) {
		t.Errorf("expected ErrCardNotFound, got: %v", err)
	}
}

func TestListCardsByUser(t *testing.T) {
	now := time.Now()
	cardRepo := &mockCardRepo{
		cards: []*entity.CardIdentifier{
			{ID: "c1", UserID: "u1", Kind: entity.CardKindRFID, Identifier: "A", IssuedAt: now},
			{ID: "c2", UserID: "u1", Kind: entity.CardKindFace, Identifier: "B", IssuedAt: now},
		},
	}
	userRepo := &mockUserRepo{user: newTestUser()}
	uc := usecase.NewCardUsecase(cardRepo, userRepo, audit.NoopLogger{})

	cards, err := uc.ListCardsByUser(context.Background(), "u1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cards) != 2 {
		t.Errorf("expected 2 cards, got %d", len(cards))
	}
}
