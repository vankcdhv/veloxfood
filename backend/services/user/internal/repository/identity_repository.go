package repository

import (
	"context"

	"project/services/user/internal/entity"
)

// IdentityRepository persists external auth identities (Google, local).
type IdentityRepository interface {
	// GetByProviderSubject finds an identity by provider + external subject.
	GetByProviderSubject(ctx context.Context, provider entity.IdentityProvider, subject string) (*entity.Identity, error)
	// Create inserts a new identity row.
	Create(ctx context.Context, identity *entity.Identity) error
}
