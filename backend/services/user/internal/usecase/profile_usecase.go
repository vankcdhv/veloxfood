package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// ProfileBundle is the full self-service view of a user.
type ProfileBundle struct {
	User              *entity.User                `json:"user"`
	Roles             []*entity.UserRole          `json:"roles"`
	StudentProfile    *entity.StudentProfile      `json:"student_profile"`
	FacultyProfile    *entity.FacultyProfile      `json:"faculty_profile"`
	VendorMemberships []*entity.VendorMembership  `json:"vendor_memberships"`
}

// UpdateMeInput contains whitelisted self-service user fields.
// Email and phone changes are deferred to Phase 2 (require verification).
type UpdateMeInput struct {
	FullName  *string        `json:"full_name"`
	DOB       *time.Time     `json:"dob"`
	Gender    *entity.Gender `json:"gender"`
	AvatarURL *string        `json:"avatar_url"`
}

// UpdateStudentProfileInput contains self-service student profile fields.
type UpdateStudentProfileInput struct {
	// Allergies replaces the entire array; nil means no change.
	Allergies     *[]string `json:"allergies"`
	DormitoryRoom *string   `json:"dormitory_room"`
}

// UpdateFacultyProfileInput contains self-service faculty profile fields.
type UpdateFacultyProfileInput struct {
	AllowPayrollDeduction *bool `json:"allow_payroll_deduction"`
}

// ProfileUsecase handles self-service profile operations.
type ProfileUsecase interface {
	GetMe(ctx context.Context, userID string) (*ProfileBundle, error)
	UpdateMe(ctx context.Context, userID string, input UpdateMeInput) (*entity.User, error)
	UpdateStudentProfile(ctx context.Context, userID string, input UpdateStudentProfileInput) (*entity.StudentProfile, error)
	UpdateFacultyProfile(ctx context.Context, userID string, input UpdateFacultyProfileInput) (*entity.FacultyProfile, error)
}

type profileUsecase struct {
	userRepo       repository.UserRepository
	studentRepo    repository.StudentProfileRepository
	facultyRepo    repository.FacultyProfileRepository
	membershipRepo repository.VendorMembershipRepository
	rbacUC         RBACUsecase
}

// NewProfileUsecase constructs ProfileUsecase.
func NewProfileUsecase(
	userRepo repository.UserRepository,
	studentRepo repository.StudentProfileRepository,
	facultyRepo repository.FacultyProfileRepository,
	membershipRepo repository.VendorMembershipRepository,
	rbacUC RBACUsecase,
) ProfileUsecase {
	return &profileUsecase{
		userRepo:       userRepo,
		studentRepo:    studentRepo,
		facultyRepo:    facultyRepo,
		membershipRepo: membershipRepo,
		rbacUC:         rbacUC,
	}
}

// GetMe loads user + roles + student + faculty + vendor memberships concurrently.
func (uc *profileUsecase) GetMe(ctx context.Context, userID string) (*ProfileBundle, error) {
	var (
		user            *entity.User
		roles           []*entity.UserRole
		memberships     []*entity.VendorMembership
		userErr         error
		rolesErr        error
		membershipsErr  error
		wg              sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		user, userErr = uc.userRepo.GetByID(ctx, userID)
	}()

	go func() {
		defer wg.Done()
		roles, rolesErr = uc.rbacUC.ListUserRoles(ctx, userID)
	}()

	go func() {
		defer wg.Done()
		if uc.membershipRepo != nil {
			memberships, membershipsErr = uc.membershipRepo.ListByUser(ctx, userID)
		}
	}()

	wg.Wait()

	if userErr != nil {
		return nil, ErrUserNotFound
	}
	if rolesErr != nil {
		slog.WarnContext(ctx, "get me: failed to load roles", "user_id", userID, "err", rolesErr)
		roles = []*entity.UserRole{}
	}
	if membershipsErr != nil {
		slog.WarnContext(ctx, "get me: failed to load vendor memberships", "user_id", userID, "err", membershipsErr)
		memberships = []*entity.VendorMembership{}
	}
	if memberships == nil {
		memberships = []*entity.VendorMembership{}
	}

	// Load profiles — only if row exists; nil means user has no profile of that type.
	student, err := uc.studentRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.WarnContext(ctx, "get me: student profile load failed", "user_id", userID, "err", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		student = nil
	}

	faculty, err := uc.facultyRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		slog.WarnContext(ctx, "get me: faculty profile load failed", "user_id", userID, "err", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		faculty = nil
	}

	return &ProfileBundle{
		User:              user,
		Roles:             roles,
		StudentProfile:    student,
		FacultyProfile:    faculty,
		VendorMemberships: memberships,
	}, nil
}

// UpdateMe applies whitelisted user field updates.
func (uc *profileUsecase) UpdateMe(ctx context.Context, userID string, input UpdateMeInput) (*entity.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	changed := false
	if input.FullName != nil {
		if *input.FullName == "" {
			return nil, ErrFullNameRequired
		}
		user.FullName = *input.FullName
		changed = true
	}
	if input.DOB != nil {
		user.DOB = input.DOB
		changed = true
	}
	if input.Gender != nil {
		if !isValidGender(*input.Gender) {
			return nil, ErrInvalidGender
		}
		user.Gender = input.Gender
		changed = true
	}
	if input.AvatarURL != nil {
		user.AvatarURL = input.AvatarURL
		changed = true
	}

	if !changed {
		return user, nil
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		slog.ErrorContext(ctx, "update me: persist failed", "user_id", userID, "err", err)
		return nil, ErrUpdateUser
	}
	slog.InfoContext(ctx, "user self-updated", "user_id", userID)
	return user, nil
}

// UpdateStudentProfile updates self-service fields on student_profiles.
// Creates profile row via Upsert if it does not yet exist (requires StudentCode to be set).
func (uc *profileUsecase) UpdateStudentProfile(ctx context.Context, userID string, input UpdateStudentProfileInput) (*entity.StudentProfile, error) {
	profile, err := uc.studentRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load student profile: %w", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileNotFound
	}

	if input.Allergies != nil {
		if len(*input.Allergies) > 10 {
			return nil, ErrAllergiesLimit
		}
		raw, merr := json.Marshal(*input.Allergies)
		if merr != nil {
			return nil, ErrProfileUpdateFailed
		}
		rawMsg := json.RawMessage(raw)
		profile.Allergies = &rawMsg
	}
	if input.DormitoryRoom != nil {
		profile.DormitoryRoom = input.DormitoryRoom
	}

	if err := uc.studentRepo.Upsert(ctx, profile); err != nil {
		slog.ErrorContext(ctx, "update student profile: upsert failed", "user_id", userID, "err", err)
		return nil, ErrProfileUpdateFailed
	}
	slog.InfoContext(ctx, "student profile updated", "user_id", userID)
	return profile, nil
}

// UpdateFacultyProfile updates self-service fields on faculty_profiles.
func (uc *profileUsecase) UpdateFacultyProfile(ctx context.Context, userID string, input UpdateFacultyProfileInput) (*entity.FacultyProfile, error) {
	profile, err := uc.facultyRepo.GetByUserID(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("load faculty profile: %w", err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrProfileNotFound
	}

	if input.AllowPayrollDeduction != nil {
		profile.AllowPayrollDeduction = *input.AllowPayrollDeduction
	}

	if err := uc.facultyRepo.Upsert(ctx, profile); err != nil {
		slog.ErrorContext(ctx, "update faculty profile: upsert failed", "user_id", userID, "err", err)
		return nil, ErrProfileUpdateFailed
	}
	slog.InfoContext(ctx, "faculty profile updated", "user_id", userID)
	return profile, nil
}

func isValidGender(g entity.Gender) bool {
	return g == entity.GenderMale || g == entity.GenderFemale || g == entity.GenderOther
}
