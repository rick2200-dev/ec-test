package domain

import (
	"time"

	"github.com/google/uuid"
)

// PlatformAdminRole represents a platform administrator's privilege level
// within a tenant. Higher roles encompass the privileges of lower roles.
type PlatformAdminRole string

const (
	PlatformAdminRoleSupport    PlatformAdminRole = "support"
	PlatformAdminRoleAdmin      PlatformAdminRole = "admin"
	PlatformAdminRoleSuperAdmin PlatformAdminRole = "super_admin"
)

// Rank returns the privilege rank of the role. Higher is more privileged.
// Unknown roles return 0.
func (r PlatformAdminRole) Rank() int {
	switch r {
	case PlatformAdminRoleSuperAdmin:
		return 3
	case PlatformAdminRoleAdmin:
		return 2
	case PlatformAdminRoleSupport:
		return 1
	}
	return 0
}

// AtLeast reports whether the role is at least as privileged as min.
func (r PlatformAdminRole) AtLeast(min PlatformAdminRole) bool {
	return r.Rank() >= min.Rank()
}

// Valid reports whether the role is one of the known values.
func (r PlatformAdminRole) Valid() bool {
	return r.Rank() > 0
}

// PlatformAdmin represents a platform administrator scoped to a tenant.
type PlatformAdmin struct {
	ID          uuid.UUID         `json:"id"`
	TenantID    uuid.UUID         `json:"tenant_id"`
	Auth0UserID string            `json:"auth0_user_id"`
	Role        PlatformAdminRole `json:"role"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
