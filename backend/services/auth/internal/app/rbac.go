package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/port"
)

// RBACService owns all role-management concerns: seller team membership,
// platform admin roles, and the audit trail. It also exposes the two
// read-only role lookups the gateway hits on every request.
type RBACService struct {
	db             port.TxRunner
	sellerUsers    port.SellerUserStore
	platformAdmins port.PlatformAdminStore
	rbacAudit      port.RBACAuditStore
}

// NewRBACService constructs an RBACService.
func NewRBACService(
	db port.TxRunner,
	sellerUsers port.SellerUserStore,
	platformAdmins port.PlatformAdminStore,
	rbacAudit port.RBACAuditStore,
) *RBACService {
	return &RBACService{
		db:             db,
		sellerUsers:    sellerUsers,
		platformAdmins: platformAdmins,
		rbacAudit:      rbacAudit,
	}
}

// ============================================================================
// Role lookup (read-only) — used by the gateway's authorization layer.
// ============================================================================

// LookupSellerRole returns the role of the given Auth0 user in a seller
// organization, or a zero string if the user is not a member of that seller.
func (s *RBACService) LookupSellerRole(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error) {
	su, err := s.sellerUsers.GetByAuth0ID(ctx, sellerID, auth0UserID)
	if err != nil {
		return "", apperrors.Internal("failed to lookup seller role", err)
	}
	if su == nil {
		return "", nil
	}
	return su.Role, nil
}

// LookupPlatformAdminRole returns the role of the given Auth0 user as a
// platform admin, or a zero string if the user is not a platform admin.
func (s *RBACService) LookupPlatformAdminRole(ctx context.Context, auth0UserID string) (domain.PlatformAdminRole, error) {
	pa, err := s.platformAdmins.GetByAuth0ID(ctx, auth0UserID)
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
func (s *RBACService) ListSellerTeam(ctx context.Context, sellerID uuid.UUID) ([]domain.SellerUser, error) {
	users, err := s.sellerUsers.ListBySeller(ctx, sellerID)
	if err != nil {
		return nil, apperrors.Internal("failed to list seller team", err)
	}
	return users, nil
}

// AddSellerUser grants a new Auth0 user a role in a seller organization.
// Only owners may add new users. The new role must not be owner — use
// TransferSellerOwnership to change the owner.
func (s *RBACService) AddSellerUser(ctx context.Context, sellerID uuid.UUID, newAuth0UserID string, role domain.SellerUserRole) (*domain.SellerUser, error) {
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
	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		// Only owners may add users.
		if err := requireSellerRoleAtLeast(txCtx, s.sellerUsers, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		// Reject if the target already belongs to the seller (UNIQUE would catch
		// it too, but we surface a nicer 409).
		existing, lookupErr := s.sellerUsers.GetByAuth0ID(txCtx, sellerID, newAuth0UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if existing != nil {
			return apperrors.Conflict("user already belongs to this seller")
		}

		created = domain.SellerUser{
			SellerID:    sellerID,
			Auth0UserID: newAuth0UserID,
			Role:        role,
		}
		if err := s.sellerUsers.Create(txCtx, &created); err != nil {
			return err
		}

		scopeID := sellerID
		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
func (s *RBACService) UpdateSellerUserRole(ctx context.Context, sellerID, targetID uuid.UUID, newRole domain.SellerUserRole) error {
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

	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		if err := requireSellerRoleAtLeast(txCtx, s.sellerUsers, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		target, err := s.sellerUsers.GetByID(txCtx, targetID)
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
			owners, err := s.sellerUsers.CountByRole(txCtx, sellerID, domain.SellerUserRoleOwner)
			if err != nil {
				return err
			}
			if owners <= 1 {
				return domain.ErrLastOwner
			}
		}

		if err := s.sellerUsers.UpdateRole(txCtx, targetID, newRole); err != nil {
			return err
		}

		scopeID := sellerID
		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
func (s *RBACService) RemoveSellerUser(ctx context.Context, sellerID, targetID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		if err := requireSellerRoleAtLeast(txCtx, s.sellerUsers, sellerID, tc.UserID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		target, err := s.sellerUsers.GetByID(txCtx, targetID)
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
			owners, err := s.sellerUsers.CountByRole(txCtx, sellerID, domain.SellerUserRoleOwner)
			if err != nil {
				return err
			}
			if owners <= 1 {
				return domain.ErrLastOwner
			}
		}

		if err := s.sellerUsers.Delete(txCtx, targetID); err != nil {
			return err
		}

		scopeID := sellerID
		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
func (s *RBACService) TransferSellerOwnership(ctx context.Context, sellerID, newOwnerID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		// Actor must be the current owner.
		actor, err := s.sellerUsers.GetByAuth0ID(txCtx, sellerID, tc.UserID)
		if err != nil {
			return err
		}
		if actor == nil || actor.Role != domain.SellerUserRoleOwner {
			return domain.ErrInsufficientRole
		}

		target, err := s.sellerUsers.GetByID(txCtx, newOwnerID)
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
		if err := s.sellerUsers.UpdateRole(txCtx, actor.ID, domain.SellerUserRoleAdmin); err != nil {
			return err
		}
		if err := s.sellerUsers.UpdateRole(txCtx, target.ID, domain.SellerUserRoleOwner); err != nil {
			return err
		}

		scopeID := sellerID
		if err := s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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

// requireSellerRoleAtLeast returns ErrInsufficientRole if the actor does
// not have at least the minimum role in the seller organization.
//
// Package-level helper (not a method) so CredentialService — which also
// gatekeeps on seller roles — can reuse it without depending on RBACService.
func requireSellerRoleAtLeast(ctx context.Context, store port.SellerUserStore, sellerID uuid.UUID, actorAuth0UserID string, min domain.SellerUserRole) error {
	role, err := store.CheckRole(ctx, sellerID, actorAuth0UserID)
	if err != nil {
		return err
	}
	if role == "" || !role.AtLeast(min) {
		return domain.ErrInsufficientRole
	}
	return nil
}

// ============================================================================
// Platform admin management
// ============================================================================

// ListPlatformAdmins returns all platform admins.
func (s *RBACService) ListPlatformAdmins(ctx context.Context) ([]domain.PlatformAdmin, error) {
	admins, err := s.platformAdmins.List(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list platform admins", err)
	}
	return admins, nil
}

// GrantPlatformAdmin grants a new Auth0 user a platform admin role.
// Only super_admins may call this.
func (s *RBACService) GrantPlatformAdmin(ctx context.Context, newAuth0UserID string, role domain.PlatformAdminRole) (*domain.PlatformAdmin, error) {
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
	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.requirePlatformAdminAtLeast(txCtx, tc.UserID, domain.PlatformAdminRoleSuperAdmin); err != nil {
			return err
		}

		existing, lookupErr := s.platformAdmins.GetByAuth0ID(txCtx, newAuth0UserID)
		if lookupErr != nil {
			return lookupErr
		}
		if existing != nil {
			return apperrors.Conflict("user is already a platform admin")
		}

		created = domain.PlatformAdmin{
			Auth0UserID: newAuth0UserID,
			Role:        role,
		}
		if err := s.platformAdmins.Create(txCtx, &created); err != nil {
			return err
		}

		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
	slog.Info("platform admin granted", "target", newAuth0UserID, "role", role, "actor", tc.UserID)
	return &created, nil
}

// UpdatePlatformAdminRole changes the role of an existing platform admin.
// Only super_admins may call this. Actors cannot change their own role, and
// the last super_admin cannot be demoted.
func (s *RBACService) UpdatePlatformAdminRole(ctx context.Context, targetID uuid.UUID, newRole domain.PlatformAdminRole) error {
	if !newRole.Valid() {
		return apperrors.BadRequest("invalid role")
	}
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.requirePlatformAdminAtLeast(txCtx, tc.UserID, domain.PlatformAdminRoleSuperAdmin); err != nil {
			return err
		}
		target, err := s.platformAdmins.GetByID(txCtx, targetID)
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
			supers, err := s.platformAdmins.CountByRole(txCtx, domain.PlatformAdminRoleSuperAdmin)
			if err != nil {
				return err
			}
			if supers <= 1 {
				return domain.ErrLastSuperAdmin
			}
		}

		if err := s.platformAdmins.UpdateRole(txCtx, targetID, newRole); err != nil {
			return err
		}

		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
	slog.Info("platform admin role updated", "target_id", targetID, "new_role", newRole, "actor", tc.UserID)
	return nil
}

// RevokePlatformAdmin removes a platform admin. Only super_admins may call
// this. Actors cannot revoke themselves, and the last super_admin cannot be
// revoked.
func (s *RBACService) RevokePlatformAdmin(ctx context.Context, targetID uuid.UUID) error {
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required")
	}

	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.requirePlatformAdminAtLeast(txCtx, tc.UserID, domain.PlatformAdminRoleSuperAdmin); err != nil {
			return err
		}
		target, err := s.platformAdmins.GetByID(txCtx, targetID)
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
			supers, err := s.platformAdmins.CountByRole(txCtx, domain.PlatformAdminRoleSuperAdmin)
			if err != nil {
				return err
			}
			if supers <= 1 {
				return domain.ErrLastSuperAdmin
			}
		}
		if err := s.platformAdmins.Delete(txCtx, targetID); err != nil {
			return err
		}
		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
	slog.Info("platform admin revoked", "target_id", targetID, "actor", tc.UserID)
	return nil
}

// ListRBACAuditLog returns a paginated list of RBAC audit entries.
func (s *RBACService) ListRBACAuditLog(ctx context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
	entries, total, err := s.rbacAudit.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list audit log", err)
	}
	return entries, total, nil
}

// requirePlatformAdminAtLeast returns ErrInsufficientRole if the actor is
// not at least the minimum platform admin role.
func (s *RBACService) requirePlatformAdminAtLeast(ctx context.Context, actorAuth0UserID string, min domain.PlatformAdminRole) error {
	role, err := s.platformAdmins.CheckRole(ctx, actorAuth0UserID)
	if err != nil {
		return err
	}
	if role == "" || !role.AtLeast(min) {
		return domain.ErrInsufficientRole
	}
	return nil
}

// ============================================================================
// Bootstrap
// ============================================================================

// BootstrapSuperAdmin ensures at least one super_admin exists.
// It is idempotent: if a super_admin already exists the call is a no-op.
// Used at service startup.
func (s *RBACService) BootstrapSuperAdmin(ctx context.Context, auth0UserID string) error {
	if auth0UserID == "" {
		return apperrors.BadRequest("bootstrap auth0_user_id is required")
	}
	count, err := s.platformAdmins.CountByRole(ctx, domain.PlatformAdminRoleSuperAdmin)
	if err != nil {
		return apperrors.Internal("failed to count platform admins", err)
	}
	if count > 0 {
		return nil
	}
	// Insert without an actor context (system bootstrap). Audit is logged
	// separately so the event is discoverable.
	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		pa := &domain.PlatformAdmin{
			Auth0UserID: auth0UserID,
			Role:        domain.PlatformAdminRoleSuperAdmin,
		}
		if err := s.platformAdmins.Create(txCtx, pa); err != nil {
			return err
		}
		return s.rbacAudit.Append(txCtx, &domain.RBACAuditEntry{
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
	slog.Warn("bootstrapped initial super_admin", "auth0_user_id", auth0UserID)
	return nil
}

// ============================================================================
// Error mapping
// ============================================================================

// mapRBACError converts domain sentinel errors into apperrors AppError values
// so handlers surface correct HTTP status codes. Other errors are wrapped as
// internal. Package-level so CredentialService's mapper can share the shape.
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
