package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"project/pkg/audit"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
	"project/services/user/internal/usecase"

	"gorm.io/gorm"
)

// ---- mock InvitationRepository ----

type mockInvitationRepo struct {
	mu             sync.Mutex
	invitations    []*entity.Invitation
	pendingByEmail map[string]*entity.Invitation // key: vendorID+":"+email
	// acceptWinner is set to a pending invitation; AcceptAtomic lets first caller win.
	acceptWinner *entity.Invitation
}

func (m *mockInvitationRepo) Create(_ context.Context, inv *entity.Invitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inv.ID = "inv-" + string(rune('A'+len(m.invitations)))
	m.invitations = append(m.invitations, inv)
	return nil
}

func (m *mockInvitationRepo) FindPendingByVendorAndEmail(_ context.Context, vendorID, email string) (*entity.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := vendorID + ":" + email
	if inv, ok := m.pendingByEmail[key]; ok {
		return inv, nil
	}
	return nil, gorm.ErrRecordNotFound
}

// AcceptAtomic simulates atomic DB accept: first caller with matching pending token wins.
func (m *mockInvitationRepo) AcceptAtomic(_ context.Context, tokenHash, userID string) (*entity.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acceptWinner == nil || m.acceptWinner.TokenHash != tokenHash {
		return nil, gorm.ErrRecordNotFound
	}
	if m.acceptWinner.Status != entity.InvitationStatusPending {
		// Already accepted — simulate 0 rows affected.
		return nil, gorm.ErrRecordNotFound
	}
	// Mark accepted — only one goroutine can reach this branch.
	m.acceptWinner.Status = entity.InvitationStatusAccepted
	m.acceptWinner.AcceptedByUserID = &userID
	clone := *m.acceptWinner
	return &clone, nil
}

func (m *mockInvitationRepo) GetByID(_ context.Context, _ string) (*entity.Invitation, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockInvitationRepo) ListByVendor(_ context.Context, _ string, _ *entity.InvitationStatus) ([]*entity.Invitation, error) {
	return nil, nil
}
func (m *mockInvitationRepo) Revoke(_ context.Context, _ string) error { return nil }

// ---- mock VendorMembershipRepository ----

type mockVendorMembershipRepo struct {
	mu      sync.Mutex
	members []*entity.VendorMembership
}

func (m *mockVendorMembershipRepo) Create(_ context.Context, mem *entity.VendorMembership) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.members = append(m.members, mem)
	return nil
}
func (m *mockVendorMembershipRepo) GetByUserAndVendor(_ context.Context, userID, vendorID string) (*entity.VendorMembership, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, mem := range m.members {
		if mem.UserID == userID && mem.VendorID == vendorID {
			return mem, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVendorMembershipRepo) ListByVendor(_ context.Context, _ string, _ *entity.VendorMembershipStatus) ([]*entity.VendorMembership, error) {
	return nil, nil
}
func (m *mockVendorMembershipRepo) ListByUser(_ context.Context, _ string) ([]*entity.VendorMembership, error) {
	return nil, nil
}
func (m *mockVendorMembershipRepo) UpdateStatus(_ context.Context, _ string, _ entity.VendorMembershipStatus) error {
	return nil
}
func (m *mockVendorMembershipRepo) UpsertActiveMembership(_ context.Context, mem *entity.VendorMembership) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.members = append(m.members, mem)
	return nil
}
func (m *mockVendorMembershipRepo) CountOwners(_ context.Context, _ string) (int, error) {
	return 1, nil
}

// ---- mock RoleRepository ----

type mockVendorRoleRepo struct {
	byCode map[string]*entity.Role
}

func newTestVendorRoleRepo() *mockVendorRoleRepo {
	return &mockVendorRoleRepo{
		byCode: map[string]*entity.Role{
			entity.RoleCodeVendorOwner:        {ID: "role-owner", Code: entity.RoleCodeVendorOwner},
			entity.RoleCodeVendorStaffKitchen: {ID: "role-kitchen", Code: entity.RoleCodeVendorStaffKitchen},
			entity.RoleCodeVendorStaffCashier: {ID: "role-cashier", Code: entity.RoleCodeVendorStaffCashier},
		},
	}
}

func (m *mockVendorRoleRepo) GetByCode(_ context.Context, code string) (*entity.Role, error) {
	if r, ok := m.byCode[code]; ok {
		return r, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVendorRoleRepo) List(_ context.Context, _ repository.RoleFilter) ([]*entity.Role, error) {
	return nil, nil
}
func (m *mockVendorRoleRepo) GetByID(_ context.Context, _ string) (*entity.Role, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockVendorRoleRepo) Create(_ context.Context, _ *entity.Role) error { return nil }
func (m *mockVendorRoleRepo) Update(_ context.Context, _ *entity.Role) error { return nil }
func (m *mockVendorRoleRepo) Delete(_ context.Context, _ string) error       { return nil }
func (m *mockVendorRoleRepo) ListPermissions(_ context.Context, _ string) ([]*entity.Permission, error) {
	return nil, nil
}
func (m *mockVendorRoleRepo) AssignPermission(_ context.Context, _, _, _ string) error { return nil }
func (m *mockVendorRoleRepo) RevokePermission(_ context.Context, _, _ string) error    { return nil }

// ---- mock OutboxRepository (for vendor tests, uses *gorm.DB sig) ----

type mockVendorOutboxRepo struct {
	mu     sync.Mutex
	events []*entity.OutboxEvent
	err    error
}

func (r *mockVendorOutboxRepo) Append(_ context.Context, _ *gorm.DB, evt *entity.OutboxEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.events = append(r.events, evt)
	return nil
}

func (r *mockVendorOutboxRepo) Pickup(_ context.Context, _, _ int) ([]*entity.OutboxEvent, error) {
	return nil, nil
}
func (r *mockVendorOutboxRepo) MarkPublished(_ context.Context, _ string) error { return nil }
func (r *mockVendorOutboxRepo) MarkFailed(_ context.Context, _, _ string, _, _ int) error {
	return nil
}

// ---- mock mailer ----

type mockInviteMailer struct {
	mu   sync.Mutex
	sent []string
}

func (m *mockInviteMailer) SendOTP(_ context.Context, _, _, _ string) error        { return nil }
func (m *mockInviteMailer) SendPasswordReset(_ context.Context, _, _ string) error { return nil }
func (m *mockInviteMailer) SendInvite(_ context.Context, to, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, to)
	return nil
}

// ---- helpers ----

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func strPtrV(s string) *string { return &s }

// buildTestStaffUC builds a VendorStaffUsecase with the given repos and nil DB.
// Tests that call Accept will return ErrInvitationInvalidOrExpired because Accept
// uses db.Transaction — with nil DB the TX call panics. To test Accept logic we
// exercise it through the mockInvitationRepo.AcceptAtomic path that the real
// implementation calls inside the transaction, tested via race test below.
func buildTestStaffUC(invRepo *mockInvitationRepo, memRepo *mockVendorMembershipRepo) usecase.VendorStaffUsecase {
	return usecase.NewVendorStaffUsecase(
		nil, // nil DB — only non-TX paths are called in invite tests
		invRepo,
		memRepo,
		newTestVendorRoleRepo(),
		&mockVendorOutboxRepo{},
		&mockRBACUsecase{},
		&mockInviteMailer{},
		"http://localhost:8080",
		audit.NoopLogger{},
	)
}

// ---- tests: Invite duplicate ----

func TestInviteStaff_Duplicate_Returns409(t *testing.T) {
	existing := &entity.Invitation{
		ID:           "existing-inv",
		VendorID:     "vendor-1",
		Email:        strPtrV("staff@example.com"),
		RoleInVendor: entity.RoleInVendorKitchen,
		Status:       entity.InvitationStatusPending,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	invRepo := &mockInvitationRepo{
		pendingByEmail: map[string]*entity.Invitation{
			"vendor-1:staff@example.com": existing,
		},
	}

	staffUC := buildTestStaffUC(invRepo, &mockVendorMembershipRepo{})
	_, err := staffUC.Invite(context.Background(), "owner-1", "vendor-1", usecase.InviteStaffInput{
		Email:        "staff@example.com",
		RoleInVendor: entity.RoleInVendorKitchen,
	})

	if !errors.Is(err, usecase.ErrInvitationDuplicate) {
		t.Errorf("expected ErrInvitationDuplicate, got: %v", err)
	}
}

func TestInviteStaff_InvalidRole_OWNER_Returns400(t *testing.T) {
	invRepo := &mockInvitationRepo{pendingByEmail: map[string]*entity.Invitation{}}
	staffUC := buildTestStaffUC(invRepo, &mockVendorMembershipRepo{})

	_, err := staffUC.Invite(context.Background(), "owner-1", "vendor-1", usecase.InviteStaffInput{
		Email:        "staff@example.com",
		RoleInVendor: entity.RoleInVendorOwner, // OWNER cannot be invited via invite flow
	})

	if !errors.Is(err, usecase.ErrInvalidInviteRole) {
		t.Errorf("expected ErrInvalidInviteRole, got: %v", err)
	}
}

func TestInviteStaff_Success_DBStoresHash(t *testing.T) {
	invRepo := &mockInvitationRepo{pendingByEmail: map[string]*entity.Invitation{}}
	staffUC := buildTestStaffUC(invRepo, &mockVendorMembershipRepo{})

	out, err := staffUC.Invite(context.Background(), "owner-1", "vendor-1", usecase.InviteStaffInput{
		Email:        "new@example.com",
		RoleInVendor: entity.RoleInVendorKitchen,
		VendorName:   "Test Canteen",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.InvitationID == "" {
		t.Error("invitation_id must be non-empty")
	}
	if out.ExpiresAt.IsZero() {
		t.Error("expires_at must be set")
	}
	if len(invRepo.invitations) != 1 {
		t.Fatalf("expected 1 stored invitation, got %d", len(invRepo.invitations))
	}
	if invRepo.invitations[0].TokenHash == "" {
		t.Error("DB must store token_hash, not raw token")
	}
}

// ---- test: AcceptAtomic race — only one goroutine wins ----

// TestAcceptAtomic_RaceCondition directly tests the mock's AcceptAtomic concurrency
// semantics (mirrors DB UPDATE WHERE status='pending' RETURNING *).
// Two goroutines call AcceptAtomic with the same token hash simultaneously — only one
// should get back the invitation; the second should get ErrRecordNotFound.
func TestAcceptAtomic_MockRaceCondition_OnlyOneWins(t *testing.T) {
	rawToken := "test-raw-token-abc123xyz456def789"
	hash := sha256Hex(rawToken)

	inv := &entity.Invitation{
		ID:           "inv-race-1",
		VendorID:     "vendor-race",
		RoleInVendor: entity.RoleInVendorKitchen,
		Status:       entity.InvitationStatusPending,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		TokenHash:    hash,
	}
	invRepo := &mockInvitationRepo{acceptWinner: inv}

	ctx := context.Background()
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners int
		losers  int
	)

	wg.Add(2)
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			userID := "user-" + string(rune('A'+i))
			result, err := invRepo.AcceptAtomic(ctx, hash, userID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil && result != nil {
				winners++
			} else {
				losers++
			}
		}()
	}
	wg.Wait()

	if winners != 1 {
		t.Errorf("expected exactly 1 winner, got %d winners and %d losers", winners, losers)
	}
	if losers != 1 {
		t.Errorf("expected exactly 1 loser, got %d winners and %d losers", winners, losers)
	}
}
