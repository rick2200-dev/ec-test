package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type ctxKey string

const (
	tenantIDKey ctxKey = "tenant_id"
	sellerIDKey ctxKey = "seller_id"
	userIDKey   ctxKey = "user_id"
	rolesKey    ctxKey = "roles"
)

var (
	ErrNoTenantID = errors.New("tenant_id not found in context")
	ErrNoUserID   = errors.New("user_id not found in context")
)

// Context holds tenant-scoped identity information extracted from JWT.
type Context struct {
	TenantID uuid.UUID
	SellerID *uuid.UUID // nil for buyers and platform admins
	UserID   string     // Auth0 sub claim
	Roles    []string
}

// WithContext stores tenant context into the Go context.
func WithContext(ctx context.Context, tc Context) context.Context {
	ctx = context.WithValue(ctx, tenantIDKey, tc.TenantID)
	ctx = context.WithValue(ctx, userIDKey, tc.UserID)
	ctx = context.WithValue(ctx, rolesKey, tc.Roles)
	if tc.SellerID != nil {
		ctx = context.WithValue(ctx, sellerIDKey, *tc.SellerID)
	}
	return ctx
}

// FromContext extracts tenant context from the Go context.
func FromContext(ctx context.Context) (Context, error) {
	tid, ok := ctx.Value(tenantIDKey).(uuid.UUID)
	if !ok {
		return Context{}, ErrNoTenantID
	}

	uid, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return Context{}, ErrNoUserID
	}

	tc := Context{
		TenantID: tid,
		UserID:   uid,
	}

	if sid, ok := ctx.Value(sellerIDKey).(uuid.UUID); ok {
		tc.SellerID = &sid
	}

	if roles, ok := ctx.Value(rolesKey).([]string); ok {
		tc.Roles = roles
	}

	return tc, nil
}

// TenantID is a convenience function to extract just the tenant ID.
func TenantID(ctx context.Context) (uuid.UUID, error) {
	tid, ok := ctx.Value(tenantIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrNoTenantID
	}
	return tid, nil
}

// HasRole checks if the context has a specific role.
func HasRole(ctx context.Context, role string) bool {
	roles, ok := ctx.Value(rolesKey).([]string)
	if !ok {
		return false
	}
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}
