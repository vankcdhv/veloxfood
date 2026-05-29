package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/usecase"

	"gorm.io/gorm"
)

// ---- mock UserRepository (minimal, reuses fields from rbac_usecase_test.go in same package) ----

type mockUserRepo struct {
	user *entity.User
	err  error
}

func (m *mockUserRepo) Create(_ context.Context, _ *entity.User) error          { return nil }
func (m *mockUserRepo) GetByID(_ context.Context, _ string) (*entity.User, error) {
	return m.user, m.err
}
func (m *mockUserRepo) GetByEmail(_ context.Context, _ string) (*entity.User, error) { return nil, nil }
func (m *mockUserRepo) Update(_ context.Context, u *entity.User) error               { return nil }
func (m *mockUserRepo) Delete(_ context.Context, _ string) error                     { return nil }
func (m *mockUserRepo) List(_ context.Context, _, _ int) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) FindByEmailOrPhone(_ context.Context, _ string) (*entity.User, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdateStatus(_ context.Context, _ string, _ entity.UserStatus) error {
	return nil
}
func (m *mockUserRepo) IncrementFailedAttempts(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockUserRepo) ResetFailedAttempts(_ context.Context, _ string) error         { return nil }
func (m *mockUserRepo) SetLockedUntil(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockUserRepo) UpdatePassword(_ context.Context, _, _ string) error           { return nil }
func (m *mockUserRepo) GetByIDs(_ context.Context, _ []string) ([]*entity.User, error) {
	return nil, nil
}

// ---- mock StudentProfileRepository ----

type mockStudentRepo struct {
	profile *entity.StudentProfile
	err     error
	upserted *entity.StudentProfile
}

func (m *mockStudentRepo) GetByUserID(_ context.Context, _ string) (*entity.StudentProfile, error) {
	return m.profile, m.err
}
func (m *mockStudentRepo) Upsert(_ context.Context, p *entity.StudentProfile) error {
	m.upserted = p
	return nil
}

// ---- mock FacultyProfileRepository ----

type mockFacultyRepo struct {
	profile *entity.FacultyProfile
	err     error
}

func (m *mockFacultyRepo) GetByUserID(_ context.Context, _ string) (*entity.FacultyProfile, error) {
	return m.profile, m.err
}
func (m *mockFacultyRepo) Upsert(_ context.Context, _ *entity.FacultyProfile) error { return nil }

// ---- mock RBACUsecase (minimal — only ListUserRoles needed) ----

type mockRBACUsecase struct {
	roles []*entity.UserRole
	err   error
}

func (m *mockRBACUsecase) HasPermission(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRBACUsecase) HasVendorPermission(_ context.Context, _, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRBACUsecase) AssignRoleToUser(_ context.Context, _, _, _ string, _ entity.ScopeType, _ *string, _ *time.Time) error {
	return nil
}
func (m *mockRBACUsecase) RemoveRoleFromUser(_ context.Context, _, _ string, _ entity.ScopeType, _ *string) error {
	return nil
}
func (m *mockRBACUsecase) ListUserRoles(_ context.Context, _ string) ([]*entity.UserRole, error) {
	return m.roles, m.err
}
func (m *mockRBACUsecase) InvalidateCacheForUser(_ context.Context, _ string) {}
func (m *mockRBACUsecase) InvalidateCacheForRole(_ context.Context, _ string) {}

// ---- helpers ----

func newTestUser() *entity.User {
	email := "test@example.com"
	return &entity.User{
		ID:       "user-uuid-1",
		Email:    &email,
		FullName: "Test User",
		Status:   entity.UserStatusActive,
	}
}

// ---- tests ----

func TestGetMe_WithStudentProfile(t *testing.T) {
	rawAllergies := json.RawMessage(`["peanut"]`)
	student := &entity.StudentProfile{
		UserID:      "user-uuid-1",
		StudentCode: "SV001",
		Allergies:   &rawAllergies,
	}

	uc := usecase.NewProfileUsecase(
		&mockUserRepo{user: newTestUser()},
		&mockStudentRepo{profile: student},
		&mockFacultyRepo{err: gorm.ErrRecordNotFound},
		nil,
		&mockRBACUsecase{roles: []*entity.UserRole{}},
	)

	bundle, err := uc.GetMe(context.Background(), "user-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bundle.User == nil {
		t.Fatal("expected user in bundle")
	}
	if bundle.StudentProfile == nil {
		t.Fatal("expected student profile in bundle")
	}
	if bundle.FacultyProfile != nil {
		t.Error("expected faculty profile to be nil")
	}
	if bundle.StudentProfile.StudentCode != "SV001" {
		t.Errorf("expected student_code=SV001, got %s", bundle.StudentProfile.StudentCode)
	}
}

func TestGetMe_NoStudentProfile(t *testing.T) {
	uc := usecase.NewProfileUsecase(
		&mockUserRepo{user: newTestUser()},
		&mockStudentRepo{err: gorm.ErrRecordNotFound},
		&mockFacultyRepo{err: gorm.ErrRecordNotFound},
		nil,
		&mockRBACUsecase{},
	)

	bundle, err := uc.GetMe(context.Background(), "user-uuid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bundle.StudentProfile != nil {
		t.Error("expected student profile to be nil when record not found")
	}
}

func TestGetMe_UserNotFound(t *testing.T) {
	uc := usecase.NewProfileUsecase(
		&mockUserRepo{err: errors.New("not found")},
		&mockStudentRepo{},
		&mockFacultyRepo{},
		nil,
		&mockRBACUsecase{},
	)

	_, err := uc.GetMe(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected error when user not found")
	}
}

func TestUpdateStudentProfile_AllergiesLimit(t *testing.T) {
	rawAllergies := json.RawMessage(`[]`)
	student := &entity.StudentProfile{
		UserID:      "user-uuid-1",
		StudentCode: "SV001",
		Allergies:   &rawAllergies,
	}

	uc := usecase.NewProfileUsecase(
		&mockUserRepo{user: newTestUser()},
		&mockStudentRepo{profile: student},
		&mockFacultyRepo{},
		nil,
		&mockRBACUsecase{},
	)

	tooMany := make([]string, 11)
	for i := range tooMany {
		tooMany[i] = "item"
	}
	_, err := uc.UpdateStudentProfile(context.Background(), "user-uuid-1", usecase.UpdateStudentProfileInput{
		Allergies: &tooMany,
	})
	if err == nil {
		t.Fatal("expected error for >10 allergies")
	}
	if err.Error() != "allergies must not exceed 10 elements" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestUpdateStudentProfile_DormitoryRoom(t *testing.T) {
	rawAllergies := json.RawMessage(`[]`)
	studentMock := &mockStudentRepo{
		profile: &entity.StudentProfile{
			UserID:      "user-uuid-1",
			StudentCode: "SV001",
			Allergies:   &rawAllergies,
		},
	}

	uc := usecase.NewProfileUsecase(
		&mockUserRepo{user: newTestUser()},
		studentMock,
		&mockFacultyRepo{},
		nil,
		&mockRBACUsecase{},
	)

	room := "A-101"
	p, err := uc.UpdateStudentProfile(context.Background(), "user-uuid-1", usecase.UpdateStudentProfileInput{
		DormitoryRoom: &room,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.DormitoryRoom == nil || *p.DormitoryRoom != "A-101" {
		t.Error("expected dormitory_room to be updated")
	}
}
