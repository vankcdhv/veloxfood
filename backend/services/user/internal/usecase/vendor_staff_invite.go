package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"project/pkg/audit"
	"project/pkg/mailer"
	"project/services/user/internal/entity"
	"project/services/user/internal/repository"

	"gorm.io/gorm"
)

const invitationTTL = 7 * 24 * time.Hour

type vendorStaffUsecase struct {
	db             *gorm.DB
	invitationRepo repository.InvitationRepository
	membershipRepo repository.VendorMembershipRepository
	roleRepo       repository.RoleRepository
	outboxRepo     repository.OutboxRepository
	rbacUC         RBACUsecase
	mailer         mailer.Mailer
	baseURL        string
	auditLogger    audit.Logger
}

// NewVendorStaffUsecase constructs VendorStaffUsecase.
func NewVendorStaffUsecase(
	db *gorm.DB,
	invitationRepo repository.InvitationRepository,
	membershipRepo repository.VendorMembershipRepository,
	roleRepo repository.RoleRepository,
	outboxRepo repository.OutboxRepository,
	rbacUC RBACUsecase,
	m mailer.Mailer,
	baseURL string,
	auditLogger audit.Logger,
) VendorStaffUsecase {
	return &vendorStaffUsecase{
		db:             db,
		invitationRepo: invitationRepo,
		membershipRepo: membershipRepo,
		roleRepo:       roleRepo,
		outboxRepo:     outboxRepo,
		rbacUC:         rbacUC,
		mailer:         m,
		baseURL:        baseURL,
		auditLogger:    auditLogger,
	}
}

// Invite creates an invitation token and sends it via email.
func (uc *vendorStaffUsecase) Invite(ctx context.Context, ownerUserID, vendorID string, input InviteStaffInput) (*InviteStaffOutput, error) {
	if input.Email == "" {
		return nil, ErrEmailRequired
	}
	if !isValidInviteRole(input.RoleInVendor) {
		return nil, ErrInvalidInviteRole
	}

	// Dedup: reject if a pending invitation for this vendor+email already exists.
	existing, err := uc.invitationRepo.FindPendingByVendorAndEmail(ctx, vendorID, input.Email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check duplicate invitation: %w", err)
	}
	if existing != nil {
		return nil, ErrInvitationDuplicate
	}

	// Generate 32-byte URL-safe token; store only SHA-256 hash in DB.
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	tokenRaw := base64.RawURLEncoding.EncodeToString(rawBytes)
	hashBytes := sha256.Sum256([]byte(tokenRaw))
	tokenHash := hex.EncodeToString(hashBytes[:])

	expiresAt := time.Now().Add(invitationTTL)
	inv := &entity.Invitation{
		VendorID:     vendorID,
		Email:        &input.Email,
		RoleInVendor: input.RoleInVendor,
		TokenHash:    tokenHash,
		ExpiresAt:    expiresAt,
		Status:       entity.InvitationStatusPending,
	}

	if err := uc.invitationRepo.Create(ctx, inv); err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}

	inviteLink := fmt.Sprintf("%s/invitations/accept?token=%s", uc.baseURL, tokenRaw)
	vendorName := input.VendorName
	if vendorName == "" {
		vendorName = vendorID
	}

	if err := uc.mailer.SendInvite(ctx, input.Email, inviteLink, vendorName); err != nil {
		// Non-fatal: log and continue — invitation row is already persisted.
		slog.WarnContext(ctx, "invite: send email failed", "invitation_id", inv.ID, "to", input.Email, "err", err)
	}

	slog.InfoContext(ctx, "invitation created", "invitation_id", inv.ID, "vendor_id", vendorID, "email", input.Email)
	_ = uc.auditLogger.Record(ctx, audit.Entry{
		ActorUserID: &ownerUserID,
		Action:      "vendor.invite_sent",
		TargetType:  "invitation",
		TargetID:    &inv.ID,
		Payload:     map[string]string{"vendor_id": vendorID, "email": input.Email, "role": string(input.RoleInVendor)},
	})
	return &InviteStaffOutput{
		InvitationID: inv.ID,
		ExpiresAt:    expiresAt,
	}, nil
}

// isValidInviteRole checks that only staff roles (not OWNER) can be invited.
func isValidInviteRole(r entity.RoleInVendor) bool {
	switch r {
	case entity.RoleInVendorManager,
		entity.RoleInVendorKitchen,
		entity.RoleInVendorCashier:
		return true
	}
	return false
}
