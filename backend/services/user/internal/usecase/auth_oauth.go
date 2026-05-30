package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"project/pkg/audit"
	authjwt "project/pkg/auth/jwt"
	"project/pkg/auth/store"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

// GoogleLoginInput carries the verified Google profile from the OAuth callback.
type GoogleLoginInput struct {
	Subject  string // Google `sub` — stable unique id
	Email    string
	FullName string
}

// OAuthUsecase handles social-login (find-or-create user + issue tokens).
type OAuthUsecase interface {
	GoogleLogin(ctx context.Context, in GoogleLoginInput) (authjwt.TokenPair, error)
}

type oauthUsecase struct {
	db           *gorm.DB
	userRepo     repository.UserRepository
	identityRepo repository.IdentityRepository
	roleRepo     repository.RoleRepository
	rbacUC       RBACUsecase
	jwtSvc       authjwt.JWTService
	store        store.AuthStore
	auditLogger  audit.Logger
}

// NewOAuthUsecase constructs OAuthUsecase.
func NewOAuthUsecase(
	db *gorm.DB,
	userRepo repository.UserRepository,
	identityRepo repository.IdentityRepository,
	roleRepo repository.RoleRepository,
	rbacUC RBACUsecase,
	jwtSvc authjwt.JWTService,
	authStore store.AuthStore,
	auditLogger audit.Logger,
) OAuthUsecase {
	return &oauthUsecase{
		db:           db,
		userRepo:     userRepo,
		identityRepo: identityRepo,
		roleRepo:     roleRepo,
		rbacUC:       rbacUC,
		jwtSvc:       jwtSvc,
		store:        authStore,
		auditLogger:  auditLogger,
	}
}

func (uc *oauthUsecase) GoogleLogin(ctx context.Context, in GoogleLoginInput) (authjwt.TokenPair, error) {
	if in.Subject == "" || in.Email == "" {
		return authjwt.TokenPair{}, ErrInvalidCredentials
	}

	user, err := uc.resolveUser(ctx, in)
	if err != nil {
		return authjwt.TokenPair{}, err
	}

	if user.Status == entity.UserStatusSuspended || user.Status == entity.UserStatusDeactivated {
		return authjwt.TokenPair{}, ErrInvalidCredentials
	}

	pair, err := uc.jwtSvc.Issue(ctx, user.ID)
	if err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("issue token: %w", err)
	}
	if err := uc.store.Whitelist(ctx, pair.JTIAccess, user.ID, pair.AccessTTL); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("whitelist access: %w", err)
	}
	if err := uc.store.Whitelist(ctx, pair.JTIRefresh, user.ID, pair.RefreshTTL); err != nil {
		return authjwt.TokenPair{}, fmt.Errorf("whitelist refresh: %w", err)
	}

	slog.InfoContext(ctx, "google login", "uid", user.ID)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &user.ID,
		Action:      "auth.google_login",
		TargetType:  "user",
		TargetID:    &user.ID,
	})
	return pair, nil
}

// resolveUser finds the user behind a Google identity, linking or creating as needed.
func (uc *oauthUsecase) resolveUser(ctx context.Context, in GoogleLoginInput) (*entity.User, error) {
	// 1. Existing google identity → existing user.
	identity, err := uc.identityRepo.GetByProviderSubject(ctx, entity.IdentityProviderGoogle, in.Subject)
	if err == nil {
		return uc.userRepo.GetByID(ctx, identity.UserID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup identity: %w", err)
	}

	// 2. No identity yet — link to existing email user or create a fresh customer.
	var user *entity.User
	txErr := uc.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, ferr := uc.userRepo.FindByEmailOrPhone(ctx, in.Email)
		switch {
		case ferr == nil && existing != nil:
			user = existing
		case errors.Is(ferr, gorm.ErrRecordNotFound):
			user = &entity.User{
				FullName: in.FullName,
				Status:   entity.UserStatusActive, // Google email is pre-verified
				Email:    strPtr(in.Email),
			}
			if cerr := uc.userRepo.Create(ctx, user); cerr != nil {
				return fmt.Errorf("create user: %w", cerr)
			}
			if rerr := uc.assignCustomerRole(ctx, user.ID); rerr != nil {
				return rerr
			}
		default:
			return fmt.Errorf("find user: %w", ferr)
		}

		if ierr := uc.identityRepo.Create(ctx, &entity.Identity{
			UserID:          user.ID,
			Provider:        entity.IdentityProviderGoogle,
			ExternalSubject: in.Subject,
		}); ierr != nil {
			return fmt.Errorf("create identity: %w", ierr)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	uc.rbacUC.InvalidateCacheForUser(ctx, user.ID)
	return user, nil
}

func (uc *oauthUsecase) assignCustomerRole(ctx context.Context, userID string) error {
	role, err := uc.roleRepo.GetByCode(ctx, entity.RoleCodeCustomer)
	if err != nil {
		return fmt.Errorf("get customer role: %w", err)
	}
	return uc.rbacUC.AssignRoleToUser(ctx, userID, userID, role.ID, entity.ScopeTypeGlobal, nil, nil)
}
