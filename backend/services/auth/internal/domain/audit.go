package domain

import (
	"time"

	"github.com/google/uuid"
)

// RBACScope identifies which RBAC table the audit entry refers to.
type RBACScope string

const (
	RBACScopeSellerUser    RBACScope = "seller_user"
	RBACScopePlatformAdmin RBACScope = "platform_admin"
)

// RBACAction identifies the kind of change recorded in an audit entry.
type RBACAction string

const (
	RBACActionGrant             RBACAction = "grant"
	RBACActionRevoke            RBACAction = "revoke"
	RBACActionRoleChange        RBACAction = "role_change"
	RBACActionTransferOwnership RBACAction = "transfer_ownership"
)

// RBACAuditEntry is one row in auth_svc.rbac_audit_log.
type RBACAuditEntry struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	ActorAuth0UserID  string     `json:"actor_auth0_user_id"`
	TargetAuth0UserID string     `json:"target_auth0_user_id"`
	Scope             RBACScope  `json:"scope"`
	ScopeID           *uuid.UUID `json:"scope_id,omitempty"` // seller_id for seller_user scope
	Action            RBACAction `json:"action"`
	BeforeRole        string     `json:"before_role,omitempty"`
	AfterRole         string     `json:"after_role,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}
