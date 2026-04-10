package domain

import "errors"

// RBAC sentinel errors. Service-layer code returns these; handlers translate
// them into HTTP status codes.
var (
	// ErrLastOwner is returned when an operation would remove or demote the
	// only remaining owner of a seller.
	ErrLastOwner = errors.New("cannot remove or demote the last owner of a seller")

	// ErrLastSuperAdmin is returned when an operation would remove or demote
	// the only remaining super_admin of a tenant.
	ErrLastSuperAdmin = errors.New("cannot remove or demote the last super_admin")

	// ErrSelfRoleChange is returned when a user attempts to change or revoke
	// their own role. Role changes must be performed by a peer or higher.
	ErrSelfRoleChange = errors.New("cannot change own role")

	// ErrInsufficientRole is returned when the acting user does not have a
	// high enough role to perform the requested action.
	ErrInsufficientRole = errors.New("insufficient role for this action")

	// ErrTargetNotFound is returned when the target user of an RBAC operation
	// does not exist.
	ErrTargetNotFound = errors.New("target user not found")

	// ErrInvalidRole is returned when an unknown or invalid role value is
	// supplied.
	ErrInvalidRole = errors.New("invalid role")

	// ErrOwnerRoleByRoleChange is returned when a caller tries to promote a
	// user to owner via the regular role-change endpoint. Ownership must go
	// through the dedicated transfer-ownership endpoint.
	ErrOwnerRoleByRoleChange = errors.New("use transfer-ownership to promote to owner")
)

// Rank returns the privilege rank of a SellerUserRole. Higher is more
// privileged. Unknown roles return 0.
func (r SellerUserRole) Rank() int {
	switch r {
	case SellerUserRoleOwner:
		return 3
	case SellerUserRoleAdmin:
		return 2
	case SellerUserRoleMember:
		return 1
	}
	return 0
}

// AtLeast reports whether the role is at least as privileged as min.
func (r SellerUserRole) AtLeast(min SellerUserRole) bool {
	return r.Rank() >= min.Rank()
}

// Valid reports whether the role is one of the known values.
func (r SellerUserRole) Valid() bool {
	return r.Rank() > 0
}
