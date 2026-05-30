package usecase

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"project/services/user/internal/entity"
	"project/services/user/internal/repository"
)

// ProfileBundle is the full self-service view of a user.
type ProfileBundle struct {
	User              *entity.User               `json:"user"`
	Roles             []*entity.UserRole         `json:"roles"`
	VendorMemberships []*entity.VendorMembership `json:"vendor_memberships"`
}

// UpdateMeInput contains whitelisted self-service user fields.
// Email and phone changes are deferred to Phase 2 (require verification).
type UpdateMeInput struct {
	FullName  *string        `json:"full_name"`
	DOB       *time.Time     `json:"dob"`
	Gender    *entity.Gender `json:"gender"`
	AvatarURL *string        `json:"avatar_url"`
}

// ProfileUsecase handles self-service profile operations.
type ProfileUsecase interface {
	GetMe(ctx context.Context, userID string) (*ProfileBundle, error)
	UpdateMe(ctx context.Context, userID string, input UpdateMeInput) (*entity.User, error)
}

type profileUsecase struct {
	userRepo       repository.UserRepository
	membershipRepo repository.VendorMembershipRepository
	rbacUC         RBACUsecase
}

// NewProfileUsecase constructs ProfileUsecase.
func NewProfileUsecase(
	userRepo repository.UserRepository,
	membershipRepo repository.VendorMembershipRepository,
	rbacUC RBACUsecase,
) ProfileUsecase {
	return &profileUsecase{
		userRepo:       userRepo,
		membershipRepo: membershipRepo,
		rbacUC:         rbacUC,
	}
}

// GetMe loads user + roles + vendor memberships concurrently.
func (uc *profileUsecase) GetMe(ctx context.Context, userID string) (*ProfileBundle, error) {
	var (
		user           *entity.User
		roles          []*entity.UserRole
		memberships    []*entity.VendorMembership
		userErr        error
		rolesErr       error
		membershipsErr error
		wg             sync.WaitGroup
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

	return &ProfileBundle{
		User:              user,
		Roles:             roles,
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

func isValidGender(g entity.Gender) bool {
	return g == entity.GenderMale || g == entity.GenderFemale || g == entity.GenderOther
}
