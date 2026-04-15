package app

import (
	"github.com/Riku-KANO/ec-test/services/auth/internal/port"
)

// AuthService is a composite facade over the three domain-specific services
// (identity / rbac / credential). Subscription was extracted into its own
// service in Phase 2.
//
// Method sets are promoted via pointer embedding, so *AuthService satisfies
// all three use-case interfaces without any hand-written delegation.
type AuthService struct {
	*IdentityService
	*RBACService
	*CredentialService
}

// NewAuthService constructs the composite facade.
func NewAuthService(
	db port.TxRunner,
	sellers port.SellerStore,
	sellerUsers port.SellerUserStore,
	platformAdmins port.PlatformAdminStore,
	rbacAudit port.RBACAuditStore,
	apiTokens port.APITokenStore,
) *AuthService {
	return &AuthService{
		IdentityService:   NewIdentityService(db, sellers, sellerUsers),
		RBACService:       NewRBACService(db, sellerUsers, platformAdmins, rbacAudit),
		CredentialService: NewCredentialService(db, apiTokens, sellerUsers),
	}
}
