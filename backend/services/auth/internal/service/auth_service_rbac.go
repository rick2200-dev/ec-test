package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Riku-KANO/ec-test/pkg/database"
	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// ============================================================================
// Role lookup (read-only) — used by the gateway's authorization layer.
// ============================================================================

// LookupSellerRole returns the role of the given Auth0 user in a seller
// organization, or a zero string if the user is not a member of that seller.
func (s *AuthService) LookupSellerRole(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error) {
	su, err := s.sellerUsers.GetByAuth0ID(ctx, tenantID, sellerID, auth0UserID)
	if err != nil {
		return "", apperrors.Internal("failed to lookup seller role", err)
	}
	if su == nil {
		return "", nil
	}
	return su.Role, nil
}

// LookupPlatformAdminRole returns the role of the given Auth0 user as a
// platform admin in the tenant, or a zero string if the user is not a
// platform admin.
func (s *AuthService) LookupPlatformAdminRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error) {
	pa, err := s.platformAdmins.GetByAuth0ID(ctx, tenantID, auth0UserID)
	if err != nil {
		return "", apperrors.Internal("failed to lookup platform admin role", err)
	}
	if pa == nil {
		return "", nil
	}
	return pa.Role, nil
}

// ============================================================================
// Seller team management
// ============================================================================

// ListSellerTeam returns all seller users for the given seller organization.
// The caller must have at least member role — the handler enforces this.
func (s *AuthService) ListSellerTeam(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error) {
	users, err := s.sellerUsers.ListBySeller(ctx, tenantID, sellerID)
	if err != nil {
		return nil, apperrors.Internal("failed to list seller team", err)
	}
	return users, nil
}

// AddSellerUser grants a new Auth0 user a role in a seller organization.
// Only owners may add new users. The new role must not be owner — use
// TransferSellerOwnership to change the owner.
func (s *AuthService) AddSellerUser(ctx context.Context, tenantID, sellerID uuid.UUID, newAuth0UserID string, role domain.SellerUserRole) (*domain.SellerUser, error) {
	if !role.Valid() {
		return nil, apperrors.BadRequest("invalid role")
	}
	if role == domain.SellerUserRoleOwner {
		return nil, apperrors.BadRequest("use transfer-ownership to assign owner role")
	}
	if newAuth0UserID == "" {
		return nil, apperrors.BadRequest("auth0_user_id is required")
	}

	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return nil, apperrors.Unauthorized("caller identity required")
	}

	var created domain.SellerUser
	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Only owners may add users.
		if err := s.requireSellerRoleAtLeastTx(ctx, tx, tenantID, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		// Reject if the target already belongs to the seller (UNIQUE would catch
		// it too, but we surface a nicer 409).
		existing, lookupErr := s.sellerUsers.GetByAuth0ID(ctx, tenantID, sellerID, newAuth0UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if existing != nil {
			return apperrors.Conflict("user already belongs to this seller")
		}

		created = domain.SellerUser{
			TenantID:    tenantID,
			SellerID:    sellerID,
			Auth0UserID: newAuth0UserID,
			Role:        role,
		}
		if err := s.sellerUsers.CreateTx(ctx, tx, &created); err != nil {
			return err
		}

		scopeID := sellerID
		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: newAuth0UserID,
			Scope:             domain.RBACScopeSellerUser,
			ScopeID:           &scopeID,
			Action:            domain.RBACActionGrant,
			AfterRole:         string(role),
		})
	})
	if err != nil {
		return nil, mapRBACError(err, "failed to add seller user")
	}

	slog.Info("seller user added", "seller_id", sellerID, "target", newAuth0UserID, "role", role, "actor", tc.UserID)
	return &created, nil
}

// UpdateSellerUserRole changes the role of an existing seller user. Only
// owners may call this. The new role must not be owner (use transfer
// ownership). Actors cannot change their own role, and the last owner of a
// seller cannot be demoted.
func (s *AuthService) UpdateSellerUserRole(ctx context.Context, tenantID, sellerID, targetID uuid.UUID, newRole domain.SellerUserRole) error {
	if !newRole.Valid() {
		return apperrors.BadRequest("invalid role")
	}
	if newRole == domain.SellerUserRoleOwner {
		return apperrors.BadRequest("use transfer-ownership to assign owner role")
	}

	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.requireSellerRoleAtLeastTx(ctx, tx, tenantID, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		target, err := s.sellerUsers.GetByID(ctx, tenantID, targetID)
		if err != nil {
			return err
		}
		if target == nil || target.SellerID != sellerID {
			return domain.ErrTargetNotFound
		}
		if target.Auth0UserID == tc.UserID {
			return domain.ErrSelfRoleChange
		}
		if target.Role == newRole {
			// No-op; return success without audit row.
			return nil
		}
		// Safeguard: cannot demote the last owner.
		if target.Role == domain.SellerUserRoleOwner {
			var owners int
			if err := s.sellerUsers.CountByRoleTx(ctx, tx, tenantID, sellerID, domain.SellerUserRoleOwner, &owners); err != nil {
				return err
			}
			if owners <= 1 {
				return domain.ErrLastOwner
			}
		}

		if err := s.sellerUsers.UpdateRoleTx(ctx, tx, tenantID, targetID, newRole); err != nil {
			return err
		}

		scopeID := sellerID
		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: target.Auth0UserID,
			Scope:             domain.RBACScopeSellerUser,
			ScopeID:           &scopeID,
			Action:            domain.RBACActionRoleChange,
			BeforeRole:        string(target.Role),
			AfterRole:         string(newRole),
		})
	})
	if err != nil {
		return mapRBACError(err, "failed to update seller user role")
	}
	slog.Info("seller user role updated", "seller_id", sellerID, "target_id", targetID, "new_role", newRole, "actor", tc.UserID)
	return nil
}

// RemoveSellerUser removes a user from a seller organization. Only owners may
// call this. Actors cannot remove themselves, and the last owner cannot be
// removed.
func (s *AuthService) RemoveSellerUser(ctx context.Context, tenantID, sellerID, targetID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.requireSellerRoleAtLeastTx(ctx, tx, tenantID, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		target, err := s.sellerUsers.GetByID(ctx, tenantID, targetID)
		if err != nil {
			return err
		}
		if target == nil || target.SellerID != sellerID {
			return domain.ErrTargetNotFound
		}
		if target.Auth0UserID == tc.UserID {
			return domain.ErrSelfRoleChange
		}
		if target.Role == domain.SellerUserRoleOwner {
			var owners int
			if err := s.sellerUsers.CountByRoleTx(ctx, tx, tenantID, sellerID, domain.SellerUserRoleOwner, &owners); err != nil {
				return err
			}
			if owners <= 1 {
				return domain.ErrLastOwner
			}
		}

		if err := s.sellerUsers.DeleteTx(ctx, tx, tenantID, targetID); err != nil {
			return err
		}

		scopeID := sellerID
		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: target.Auth0UserID,
			Scope:             domain.RBACScopeSellerUser,
			ScopeID:           &scopeID,
			Action:            domain.RBACActionRevoke,
			BeforeRole:        string(target.Role),
		})
	})
	if err != nil {
		return mapRBACError(err, "failed to remove seller user")
	}
	slog.Info("seller user removed", "seller_id", sellerID, "target_id", targetID, "actor", tc.UserID)
	return nil
}

// TransferSellerOwnership promotes an existing admin/member to owner and
// atomically demotes the current owner to admin. Only the current owner may
// call this. The new owner must already be a member of the seller team.
func (s *AuthService) TransferSellerOwnership(ctx context.Context, tenantID, sellerID, newOwnerID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// Actor must be the current owner.
		actor, err := s.sellerUsers.GetByAuth0ID(ctx, tenantID, sellerID, tc.UserID)
		if err != nil {
			return err
		}
		if actor == nil || actor.Role != domain.SellerUserRoleOwner {
			return domain.ErrInsufficientRole
		}

		target, err := s.sellerUsers.GetByID(ctx, tenantID, newOwnerID)
		if err != nil {
			return err
		}
		if target == nil || target.SellerID != sellerID {
			return domain.ErrTargetNotFound
		}
		if target.ID == actor.ID {
			return domain.ErrSelfRoleChange
		}

		beforeTarget := target.Role

		// Demote current owner to admin first, then promote the target.
		if err := s.sellerUsers.UpdateRoleTx(ctx, tx, tenantID, actor.ID, domain.SellerUserRoleAdmin); err != nil {
			return err
		}
		if err := s.sellerUsers.UpdateRoleTx(ctx, tx, tenantID, target.ID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		scopeID := sellerID
		if err := s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: target.Auth0UserID,
			Scope:             domain.RBACScopeSellerUser,
			ScopeID:           &scopeID,
			Action:            domain.RBACActionTransferOwnership,
			BeforeRole:        string(beforeTarget),
			AfterRole:         string(domain.SellerUserRoleOwner),
		}); err != nil {
			return err
		}
		// Record the demotion of the previous owner as a separate row.
		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: actor.Auth0UserID,
			Scope:             domain.RBACScopeSellerUser,
			ScopeID:           &scopeID,
			Action:            domain.RBACActionRoleChange,
			BeforeRole:        string(domain.SellerUserRoleOwner),
			AfterRole:         string(domain.SellerUserRoleAdmin),
		})
	})
	if err != nil {
		return mapRBACError(err, "failed to transfer ownership")
	}
	slog.Info("seller ownership transferred", "seller_id", sellerID, "new_owner_id", newOwnerID, "actor", tc.UserID)
	return nil
}

// requireSellerRoleAtLeastTx returns ErrInsufficientRole if the actor does
// not have at least the minimum role in the seller organization. Must be
// called inside a tenant-scoped transaction.
func (s *AuthService) requireSellerRoleAtLeastTx(ctx context.Context, tx pgx.Tx, tenantID, sellerID uuid.UUID, actorAuth0UserID string, min domain.SellerUserRole) error {
	// We do this via the non-Tx lookup because RLS tenant_id is already set on
	// tx — but we need the same tenant session variable to be in effect. The
	// tx parameter is already inside a TenantTx so the session var is set.
	var role domain.SellerUserRole
	err := tx.QueryRow(ctx,
		`SELECT role FROM auth_svc.seller_users
		 WHERE tenant_id = $1 AND seller_id = $2 AND auth0_user_id = $3`,
		tenantID, sellerID, actorAuth0UserID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInsufficientRole
		}
		return err
	}
	if !role.AtLeast(min) {
		return domain.ErrInsufficientRole
	}
	return nil
}

// ============================================================================
// Platform admin management
// ============================================================================

// ListPlatformAdmins returns all platform admins in the tenant.
func (s *AuthService) ListPlatformAdmins(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error) {
	admins, err := s.platformAdmins.List(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal("failed to list platform admins", err)
	}
	return admins, nil
}

// GrantPlatformAdmin grants a new Auth0 user a platform admin role in the
// tenant. Only super_admins may call this.
func (s *AuthService) GrantPlatformAdmin(ctx context.Context, tenantID uuid.UUID, newAuth0UserID string, role domain.PlatformAdminRole) (*domain.PlatformAdmin, error) {
	if !role.Valid() {
		return nil, apperrors.BadRequest("invalid role")
	}
	if newAuth0UserID == "" {
		return nil, apperrors.BadRequest("auth0_user_id is required")
	}

	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return nil, apperrors.Unauthorized("caller identity required")
	}

	var created domain.PlatformAdmin
	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.requirePlatformAdminAtLeastTx(ctx, tx, tenantID, tc.UserID, domain.PlatformAdminRoleSuperAdmin); err != nil {
			return err
		}

		existing, lookupErr := s.platformAdmins.GetByAuth0ID(ctx, tenantID, newAuth0UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if existing != nil {
			return apperrors.Conflict("user is already a platform admin in this tenant")
		}

		created = domain.PlatformAdmin{
			TenantID:    tenantID,
			Auth0UserID: newAuth0UserID,
			Role:        role,
		}
		if err := s.platformAdmins.CreateTx(ctx, tx, &created); err != nil {
			return err
		}

		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: newAuth0UserID,
			Scope:             domain.RBACScopePlatformAdmin,
			Action:            domain.RBACActionGrant,
			AfterRole:         string(role),
		})
	})
	if err != nil {
		return nil, mapRBACError(err, "failed to grant platform admin")
	}
	slog.Info("platform admin granted", "tenant_id", tenantID, "target", newAuth0UserID, "role", role, "actor", tc.UserID)
	return &created, nil
}

// UpdatePlatformAdminRole changes the role of an existing platform admin.
// Only super_admins may call this. Actors cannot change their own role, and
// the last super_admin cannot be demoted.
func (s *AuthService) UpdatePlatformAdminRole(ctx context.Context, tenantID, targetID uuid.UUID, newRole domain.PlatformAdminRole) error {
	if !newRole.Valid() {
		return apperrors.BadRequest("invalid role")
	}
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.requirePlatformAdminAtLeastTx(ctx, tx, tenantID, tc.UserID, domain.PlatformAdminRoleSuperAdmin); err != nil {
			return err
		}
		target, err := s.platformAdmins.GetByID(ctx, tenantID, targetID)
		if err != nil {
			return err
		}
		if target == nil {
			return domain.ErrTargetNotFound
		}
		if target.Auth0UserID == tc.UserID {
			return domain.ErrSelfRoleChange
		}
		if target.Role == newRole {
			return nil
		}
		if target.Role == domain.PlatformAdminRoleSuperAdmin {
			var supers int
			if err := s.platformAdmins.CountByRoleTx(ctx, tx, tenantID, domain.PlatformAdminRoleSuperAdmin, &supers); err != nil {
				return err
			}
			if supers <= 1 {
				return domain.ErrLastSuperAdmin
			}
		}

		if err := s.platformAdmins.UpdateRoleTx(ctx, tx, tenantID, targetID, newRole); err != nil {
			return err
		}

		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: target.Auth0UserID,
			Scope:             domain.RBACScopePlatformAdmin,
			Action:            domain.RBACActionRoleChange,
			BeforeRole:        string(target.Role),
			AfterRole:         string(newRole),
		})
	})
	if err != nil {
		return mapRBACError(err, "failed to update platform admin role")
	}
	slog.Info("platform admin role updated", "tenant_id", tenantID, "target_id", targetID, "new_role", newRole, "actor", tc.UserID)
	return nil
}

// RevokePlatformAdmin removes a platform admin. Only super_admins may call
// this. Actors cannot revoke themselves, and the last super_admin cannot be
// revoked.
func (s *AuthService) RevokePlatformAdmin(ctx context.Context, tenantID, targetID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if err := s.requirePlatformAdminAtLeastTx(ctx, tx, tenantID, tc.UserID, domain.PlatformAdminRoleSuperAdmin); err != nil {
			return err
		}
		target, err := s.platformAdmins.GetByID(ctx, tenantID, targetID)
		if err != nil {
			return err
		}
		if target == nil {
			return domain.ErrTargetNotFound
		}
		if target.Auth0UserID == tc.UserID {
			return domain.ErrSelfRoleChange
		}
		if target.Role == domain.PlatformAdminRoleSuperAdmin {
			var supers int
			if err := s.platformAdmins.CountByRoleTx(ctx, tx, tenantID, domain.PlatformAdminRoleSuperAdmin, &supers); err != nil {
				return err
			}
			if supers <= 1 {
				return domain.ErrLastSuperAdmin
			}
		}
		if err := s.platformAdmins.DeleteTx(ctx, tx, tenantID, targetID); err != nil {
			return err
		}
		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  tc.UserID,
			TargetAuth0UserID: target.Auth0UserID,
			Scope:             domain.RBACScopePlatformAdmin,
			Action:            domain.RBACActionRevoke,
			BeforeRole:        string(target.Role),
		})
	})
	if err != nil {
		return mapRBACError(err, "failed to revoke platform admin")
	}
	slog.Info("platform admin revoked", "tenant_id", tenantID, "target_id", targetID, "actor", tc.UserID)
	return nil
}

// ListRBACAuditLog returns a paginated list of RBAC audit entries for the
// tenant.
func (s *AuthService) ListRBACAuditLog(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
	entries, total, err := s.rbacAudit.ListByTenant(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list audit log", err)
	}
	return entries, total, nil
}

// requirePlatformAdminAtLeastTx returns ErrInsufficientRole if the actor is
// not at least the minimum platform admin role in the tenant.
func (s *AuthService) requirePlatformAdminAtLeastTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, actorAuth0UserID string, min domain.PlatformAdminRole) error {
	var role domain.PlatformAdminRole
	err := tx.QueryRow(ctx,
		`SELECT role FROM auth_svc.platform_admins
		 WHERE tenant_id = $1 AND auth0_user_id = $2`,
		tenantID, actorAuth0UserID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInsufficientRole
		}
		return err
	}
	if !role.AtLeast(min) {
		return domain.ErrInsufficientRole
	}
	return nil
}

// ============================================================================
// Bootstrap
// ============================================================================

// BootstrapSuperAdmin ensures at least one super_admin exists for the given
// tenant. It is idempotent: if a super_admin already exists the call is a
// no-op. Used at service startup.
func (s *AuthService) BootstrapSuperAdmin(ctx context.Context, tenantID uuid.UUID, auth0UserID string) error {
	if auth0UserID == "" {
		return apperrors.BadRequest("bootstrap auth0_user_id is required")
	}
	count, err := s.platformAdmins.CountByRole(ctx, tenantID, domain.PlatformAdminRoleSuperAdmin)
	if err != nil {
		return apperrors.Internal("failed to count platform admins", err)
	}
	if count > 0 {
		return nil
	}
	// Insert without an actor context (system bootstrap). Audit is logged
	// separately so the event is discoverable.
	err = database.TenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		pa := &domain.PlatformAdmin{
			TenantID:    tenantID,
			Auth0UserID: auth0UserID,
			Role:        domain.PlatformAdminRoleSuperAdmin,
		}
		if err := s.platformAdmins.CreateTx(ctx, tx, pa); err != nil {
			return err
		}
		return s.rbacAudit.AppendTx(ctx, tx, &domain.RBACAuditEntry{
			TenantID:          tenantID,
			ActorAuth0UserID:  "system:bootstrap",
			TargetAuth0UserID: auth0UserID,
			Scope:             domain.RBACScopePlatformAdmin,
			Action:            domain.RBACActionGrant,
			AfterRole:         string(domain.PlatformAdminRoleSuperAdmin),
		})
	})
	if err != nil {
		return apperrors.Internal("failed to bootstrap super_admin", err)
	}
	slog.Warn("bootstrapped initial super_admin", "tenant_id", tenantID, "auth0_user_id", auth0UserID)
	return nil
}

// ============================================================================
// Error mapping
// ============================================================================

// mapRBACError converts domain sentinel errors into apperrors AppError values
// so handlers surface correct HTTP status codes. Other errors are wrapped as
// internal.
func mapRBACError(err error, internalMsg string) error {
	if err == nil {
		return nil
	}
	// Pass through pre-mapped AppErrors (from e.g. nested Conflict/BadRequest).
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, domain.ErrLastOwner),
		errors.Is(err, domain.ErrLastSuperAdmin):
		return apperrors.Conflict(err.Error())
	case errors.Is(err, domain.ErrSelfRoleChange),
		errors.Is(err, domain.ErrInsufficientRole):
		return apperrors.Forbidden(err.Error())
	case errors.Is(err, domain.ErrTargetNotFound):
		return apperrors.NotFound(err.Error())
	case errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrOwnerRoleByRoleChange):
		return apperrors.BadRequest(err.Error())
	}
	return apperrors.Internal(internalMsg, err)
}
