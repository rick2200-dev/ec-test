package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type ctxKey string

const (
	sellerIDKey ctxKey = "seller_id"
	userIDKey   ctxKey = "user_id"
	rolesKey    ctxKey = "roles"
)

var (
	ErrNoUserID = errors.New("user_id not found in context")
)

// Context holds request-scoped identity information extracted from JWT.
type Context struct {
	SellerID *uuid.UUID // nil for buyers and platform admins
	UserID   string     // Auth0 sub claim
	Roles    []string
}

// WithContext stores identity context into the Go context.
func WithContext(ctx context.Context, tc Context) context.Context {
	ctx = context.WithValue(ctx, userIDKey, tc.UserID)
	ctx = context.WithValue(ctx, rolesKey, tc.Roles)
	if tc.SellerID != nil {
		ctx = context.WithValue(ctx, sellerIDKey, *tc.SellerID)
	}
	return ctx
}

// FromContext extracts identity context from the Go context.
func FromContext(ctx context.Context) (Context, error) {
	uid, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return Context{}, ErrNoUserID
	}

	tc := Context{
		UserID: uid,
	}

	if sid, ok := ctx.Value(sellerIDKey).(uuid.UUID); ok {
		tc.SellerID = &sid
	}

	if roles, ok := ctx.Value(rolesKey).([]string); ok {
		tc.Roles = roles
	}

	return tc, nil
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
