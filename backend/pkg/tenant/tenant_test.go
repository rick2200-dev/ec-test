package tenant_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/tenant"
)

func TestWithContext_FromContext_RoundTrip(t *testing.T) {
	sid := uuid.New()
	tc := tenant.Context{
		SellerID: &sid,
		UserID:   "auth0|user1",
		Roles:    []string{"seller", "admin"},
	}

	ctx := tenant.WithContext(context.Background(), tc)
	got, err := tenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext returned error: %v", err)
	}
	if got.UserID != "auth0|user1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "auth0|user1")
	}
	if got.SellerID == nil || *got.SellerID != sid {
		t.Errorf("SellerID = %v, want %v", got.SellerID, sid)
	}
	if len(got.Roles) != 2 || got.Roles[0] != "seller" || got.Roles[1] != "admin" {
		t.Errorf("Roles = %v, want [seller admin]", got.Roles)
	}
}

func TestWithContext_NilSellerID(t *testing.T) {
	tc := tenant.Context{
		UserID: "auth0|buyer1",
		Roles:  []string{"buyer"},
	}

	ctx := tenant.WithContext(context.Background(), tc)
	got, err := tenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext returned error: %v", err)
	}
	if got.SellerID != nil {
		t.Errorf("SellerID = %v, want nil", got.SellerID)
	}
}

func TestFromContext_EmptyContext(t *testing.T) {
	_, err := tenant.FromContext(context.Background())
	if !errors.Is(err, tenant.ErrNoUserID) {
		t.Errorf("expected ErrNoUserID, got %v", err)
	}
}

func TestFromContext_MissingUserID(t *testing.T) {
	// We cannot inject a partial context with the unexported ctxKey type,
	// so we verify the empty-context path returns ErrNoUserID.
	_, err := tenant.FromContext(context.Background())
	if !errors.Is(err, tenant.ErrNoUserID) {
		t.Errorf("expected ErrNoUserID, got %v", err)
	}
}

func TestHasRole_Present(t *testing.T) {
	tc := tenant.Context{
		UserID: "auth0|u",
		Roles:  []string{"buyer", "admin"},
	}
	ctx := tenant.WithContext(context.Background(), tc)

	if !tenant.HasRole(ctx, "buyer") {
		t.Error("expected HasRole(buyer) = true")
	}
	if !tenant.HasRole(ctx, "admin") {
		t.Error("expected HasRole(admin) = true")
	}
}

func TestHasRole_Absent(t *testing.T) {
	tc := tenant.Context{
		UserID: "auth0|u",
		Roles:  []string{"buyer"},
	}
	ctx := tenant.WithContext(context.Background(), tc)

	if tenant.HasRole(ctx, "seller") {
		t.Error("expected HasRole(seller) = false")
	}
}

func TestHasRole_NoRolesInContext(t *testing.T) {
	if tenant.HasRole(context.Background(), "anything") {
		t.Error("expected HasRole = false on empty context")
	}
}

func TestHasRole_EmptyRoles(t *testing.T) {
	tc := tenant.Context{
		UserID: "auth0|u",
		Roles:  []string{},
	}
	ctx := tenant.WithContext(context.Background(), tc)

	if tenant.HasRole(ctx, "buyer") {
		t.Error("expected HasRole = false with empty roles slice")
	}
}

func TestFromContext_NilRoles(t *testing.T) {
	tc := tenant.Context{
		UserID: "auth0|u",
	}
	ctx := tenant.WithContext(context.Background(), tc)
	got, err := tenant.FromContext(ctx)
	if err != nil {
		t.Fatalf("FromContext returned error: %v", err)
	}
	if got.Roles != nil {
		t.Errorf("Roles = %v, want nil", got.Roles)
	}
}
