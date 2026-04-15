package app_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/app"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// ============================================================================
// Function-field mock types
// ============================================================================

type mockTxRunner struct {
	RunTxFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTxRunner) RunTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.RunTxFn != nil {
		return m.RunTxFn(ctx, fn)
	}
	return fn(ctx)
}

type mockSellerStore struct {
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.Seller, error)
	GetBySlugFn    func(ctx context.Context, slug string) (*domain.Seller, error)
	ListFn         func(ctx context.Context, limit, offset int) ([]domain.Seller, int, error)
	UpdateStatusFn func(ctx context.Context, id uuid.UUID, status domain.SellerStatus) error
	CreateFn       func(ctx context.Context, s *domain.Seller) error
}

func (m *mockSellerStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Seller, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockSellerStore) GetBySlug(ctx context.Context, slug string) (*domain.Seller, error) {
	if m.GetBySlugFn != nil {
		return m.GetBySlugFn(ctx, slug)
	}
	return nil, nil
}
func (m *mockSellerStore) List(ctx context.Context, limit, offset int) ([]domain.Seller, int, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, limit, offset)
	}
	return nil, 0, nil
}
func (m *mockSellerStore) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SellerStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}
func (m *mockSellerStore) Create(ctx context.Context, s *domain.Seller) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, s)
	}
	return nil
}
func (m *mockSellerStore) BatchGetByIDs(_ context.Context, _ []uuid.UUID) ([]domain.Seller, error) {
	return nil, nil
}

type mockSellerUserStore struct {
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.SellerUser, error)
	GetByAuth0IDFn func(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error)
	ListBySellerFn func(ctx context.Context, sellerID uuid.UUID) ([]domain.SellerUser, error)
	CreateFn       func(ctx context.Context, su *domain.SellerUser) error
	UpdateRoleFn   func(ctx context.Context, id uuid.UUID, role domain.SellerUserRole) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	CountByRoleFn  func(ctx context.Context, sellerID uuid.UUID, role domain.SellerUserRole) (int, error)
	CheckRoleFn    func(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
}

func (m *mockSellerUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.SellerUser, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockSellerUserStore) GetByAuth0ID(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error) {
	if m.GetByAuth0IDFn != nil {
		return m.GetByAuth0IDFn(ctx, sellerID, auth0UserID)
	}
	return nil, nil
}
func (m *mockSellerUserStore) ListBySeller(ctx context.Context, sellerID uuid.UUID) ([]domain.SellerUser, error) {
	if m.ListBySellerFn != nil {
		return m.ListBySellerFn(ctx, sellerID)
	}
	return nil, nil
}
func (m *mockSellerUserStore) Create(ctx context.Context, su *domain.SellerUser) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, su)
	}
	return nil
}
func (m *mockSellerUserStore) UpdateRole(ctx context.Context, id uuid.UUID, role domain.SellerUserRole) error {
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, id, role)
	}
	return nil
}
func (m *mockSellerUserStore) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *mockSellerUserStore) CountByRole(ctx context.Context, sellerID uuid.UUID, role domain.SellerUserRole) (int, error) {
	if m.CountByRoleFn != nil {
		return m.CountByRoleFn(ctx, sellerID, role)
	}
	return 0, nil
}
func (m *mockSellerUserStore) CheckRole(ctx context.Context, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error) {
	if m.CheckRoleFn != nil {
		return m.CheckRoleFn(ctx, sellerID, auth0UserID)
	}
	return "", nil
}

type mockPlatformAdminStore struct {
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error)
	GetByAuth0IDFn func(ctx context.Context, auth0UserID string) (*domain.PlatformAdmin, error)
	ListFn         func(ctx context.Context) ([]domain.PlatformAdmin, error)
	CountByRoleFn  func(ctx context.Context, role domain.PlatformAdminRole) (int, error)
	CreateFn       func(ctx context.Context, pa *domain.PlatformAdmin) error
	UpdateRoleFn   func(ctx context.Context, id uuid.UUID, role domain.PlatformAdminRole) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	CheckRoleFn    func(ctx context.Context, auth0UserID string) (domain.PlatformAdminRole, error)
}

func (m *mockPlatformAdminStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockPlatformAdminStore) GetByAuth0ID(ctx context.Context, auth0UserID string) (*domain.PlatformAdmin, error) {
	if m.GetByAuth0IDFn != nil {
		return m.GetByAuth0IDFn(ctx, auth0UserID)
	}
	return nil, nil
}
func (m *mockPlatformAdminStore) List(ctx context.Context) ([]domain.PlatformAdmin, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx)
	}
	return nil, nil
}
func (m *mockPlatformAdminStore) CountByRole(ctx context.Context, role domain.PlatformAdminRole) (int, error) {
	if m.CountByRoleFn != nil {
		return m.CountByRoleFn(ctx, role)
	}
	return 0, nil
}
func (m *mockPlatformAdminStore) Create(ctx context.Context, pa *domain.PlatformAdmin) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, pa)
	}
	return nil
}
func (m *mockPlatformAdminStore) UpdateRole(ctx context.Context, id uuid.UUID, role domain.PlatformAdminRole) error {
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, id, role)
	}
	return nil
}
func (m *mockPlatformAdminStore) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *mockPlatformAdminStore) CheckRole(ctx context.Context, auth0UserID string) (domain.PlatformAdminRole, error) {
	if m.CheckRoleFn != nil {
		return m.CheckRoleFn(ctx, auth0UserID)
	}
	return "", nil
}

type mockRBACAuditStore struct {
	AppendFn func(ctx context.Context, e *domain.RBACAuditEntry) error
	ListFn   func(ctx context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error)
}

func (m *mockRBACAuditStore) Append(ctx context.Context, e *domain.RBACAuditEntry) error {
	if m.AppendFn != nil {
		return m.AppendFn(ctx, e)
	}
	return nil
}
func (m *mockRBACAuditStore) List(ctx context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

type mockAPITokenStore struct {
	CreateFn          func(ctx context.Context, t *domain.SellerAPIToken) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*domain.SellerAPIToken, error)
	ListBySellerFn    func(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	RevokeFn          func(ctx context.Context, id uuid.UUID, actorAuth0UserID string) error
	GetByLookupFn     func(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error)
	TouchLastUsedAtFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockAPITokenStore) Create(ctx context.Context, t *domain.SellerAPIToken) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, t)
	}
	return nil
}
func (m *mockAPITokenStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.SellerAPIToken, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockAPITokenStore) ListBySeller(ctx context.Context, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error) {
	if m.ListBySellerFn != nil {
		return m.ListBySellerFn(ctx, sellerID, limit, offset)
	}
	return nil, 0, nil
}
func (m *mockAPITokenStore) Revoke(ctx context.Context, id uuid.UUID, actorAuth0UserID string) error {
	if m.RevokeFn != nil {
		return m.RevokeFn(ctx, id, actorAuth0UserID)
	}
	return nil
}
func (m *mockAPITokenStore) GetByLookup(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error) {
	if m.GetByLookupFn != nil {
		return m.GetByLookupFn(ctx, prefix, lookup)
	}
	return nil, nil
}
func (m *mockAPITokenStore) TouchLastUsedAt(ctx context.Context, id uuid.UUID) error {
	if m.TouchLastUsedAtFn != nil {
		return m.TouchLastUsedAtFn(ctx, id)
	}
	return nil
}

// ============================================================================
// Helpers
// ============================================================================

// newService creates an AuthService with all mocks wired in. Only the mocks
// the caller explicitly sets will be non-nil; the rest use zero-value stubs.
func newService(
	db *mockTxRunner,
	sellers *mockSellerStore,
	sellerUsers *mockSellerUserStore,
	platformAdmins *mockPlatformAdminStore,
	rbacAudit *mockRBACAuditStore,
	apiTokens *mockAPITokenStore,
) *app.AuthService {
	if db == nil {
		db = &mockTxRunner{}
	}
	if sellers == nil {
		sellers = &mockSellerStore{}
	}
	if sellerUsers == nil {
		sellerUsers = &mockSellerUserStore{}
	}
	if platformAdmins == nil {
		platformAdmins = &mockPlatformAdminStore{}
	}
	if rbacAudit == nil {
		rbacAudit = &mockRBACAuditStore{}
	}
	if apiTokens == nil {
		apiTokens = &mockAPITokenStore{}
	}
	return app.NewAuthService(db, sellers, sellerUsers, platformAdmins, rbacAudit, apiTokens)
}

// requireAppError asserts the error is an *apperrors.AppError with the expected
// HTTP status code and returns it.
func requireAppError(t *testing.T, err error, wantStatus int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with status %d, got nil", wantStatus)
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperrors.AppError, got %T: %v", err, err)
	}
	if appErr.Status != wantStatus {
		t.Fatalf("status = %d, want %d (message: %s)", appErr.Status, wantStatus, appErr.Message)
	}
}

// ctxWithUser returns a context enriched with identity context for the given
// Auth0 user ID.
func ctxWithUser(userID string) context.Context {
	return tenant.WithContext(context.Background(), tenant.Context{
		UserID: userID,
	})
}

// ============================================================================
// GetSeller
// ============================================================================

func TestGetSeller_Success(t *testing.T) {
	sid := uuid.New()
	svc := newService(nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{ID: sid}, nil
		},
	}, nil, nil, nil, nil)

	got, err := svc.GetSeller(context.Background(), sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != sid {
		t.Errorf("ID = %v, want %v", got.ID, sid)
	}
}

func TestGetSeller_NotFound(t *testing.T) {
	svc := newService(nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Seller, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil)

	_, err := svc.GetSeller(context.Background(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// ListSellers
// ============================================================================

func TestListSellers_Success(t *testing.T) {
	want := []domain.Seller{{Name: "S1"}, {Name: "S2"}}
	svc := newService(nil, &mockSellerStore{
		ListFn: func(_ context.Context, limit, offset int) ([]domain.Seller, int, error) {
			return want, 2, nil
		},
	}, nil, nil, nil, nil)

	got, total, err := svc.ListSellers(context.Background(), 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Errorf("total=%d len=%d, want 2,2", total, len(got))
	}
}

// ============================================================================
// ApproveSeller
// ============================================================================

func TestApproveSeller_Success(t *testing.T) {
	sid := uuid.New()
	var statusUpdated bool
	svc := newService(nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{ID: sid, Status: domain.SellerStatusPending}, nil
		},
		UpdateStatusFn: func(_ context.Context, _ uuid.UUID, status domain.SellerStatus) error {
			if status != domain.SellerStatusApproved {
				t.Errorf("status = %v, want approved", status)
			}
			statusUpdated = true
			return nil
		},
	}, nil, nil, nil, nil)

	err := svc.ApproveSeller(context.Background(), sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !statusUpdated {
		t.Fatal("UpdateStatus was not called")
	}
}

func TestApproveSeller_NotFound(t *testing.T) {
	svc := newService(nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Seller, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil)

	err := svc.ApproveSeller(context.Background(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestApproveSeller_NotPending(t *testing.T) {
	svc := newService(nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{Status: domain.SellerStatusApproved}, nil
		},
	}, nil, nil, nil, nil)

	err := svc.ApproveSeller(context.Background(), uuid.New())
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// LookupSellerRole
// ============================================================================

func TestLookupSellerRole_Found(t *testing.T) {
	sid := uuid.New()
	svc := newService(nil, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, auth0UserID string) (*domain.SellerUser, error) {
			return &domain.SellerUser{Role: domain.SellerUserRoleOwner, Auth0UserID: auth0UserID}, nil
		},
	}, nil, nil, nil)

	role, err := svc.LookupSellerRole(context.Background(), sid, "auth0|user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != domain.SellerUserRoleOwner {
		t.Errorf("role = %q, want %q", role, domain.SellerUserRoleOwner)
	}
}

func TestLookupSellerRole_NotFound(t *testing.T) {
	svc := newService(nil, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.SellerUser, error) {
			return nil, nil
		},
	}, nil, nil, nil)

	role, err := svc.LookupSellerRole(context.Background(), uuid.New(), "auth0|nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != "" {
		t.Errorf("role = %q, want empty string", role)
	}
}

// ============================================================================
// LookupPlatformAdminRole
// ============================================================================

func TestLookupPlatformAdminRole_Found(t *testing.T) {
	svc := newService(nil, nil, nil, &mockPlatformAdminStore{
		GetByAuth0IDFn: func(_ context.Context, auth0UserID string) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{Role: domain.PlatformAdminRoleSuperAdmin}, nil
		},
	}, nil, nil)

	role, err := svc.LookupPlatformAdminRole(context.Background(), "auth0|admin1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != domain.PlatformAdminRoleSuperAdmin {
		t.Errorf("role = %q, want %q", role, domain.PlatformAdminRoleSuperAdmin)
	}
}

func TestLookupPlatformAdminRole_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, &mockPlatformAdminStore{
		GetByAuth0IDFn: func(_ context.Context, _ string) (*domain.PlatformAdmin, error) {
			return nil, nil
		},
	}, nil, nil)

	role, err := svc.LookupPlatformAdminRole(context.Background(), "auth0|nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != "" {
		t.Errorf("role = %q, want empty string", role)
	}
}

// ============================================================================
// ListSellerTeam
// ============================================================================

func TestListSellerTeam_Success(t *testing.T) {
	sid := uuid.New()
	want := []domain.SellerUser{
		{Auth0UserID: "auth0|u1", Role: domain.SellerUserRoleOwner},
		{Auth0UserID: "auth0|u2", Role: domain.SellerUserRoleMember},
	}
	svc := newService(nil, nil, &mockSellerUserStore{
		ListBySellerFn: func(_ context.Context, _ uuid.UUID) ([]domain.SellerUser, error) {
			return want, nil
		},
	}, nil, nil, nil)

	got, err := svc.ListSellerTeam(context.Background(), sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// ListPlatformAdmins
// ============================================================================

func TestListPlatformAdmins_Success(t *testing.T) {
	want := []domain.PlatformAdmin{
		{Auth0UserID: "auth0|sa1", Role: domain.PlatformAdminRoleSuperAdmin},
		{Auth0UserID: "auth0|a1", Role: domain.PlatformAdminRoleAdmin},
	}
	svc := newService(nil, nil, nil, &mockPlatformAdminStore{
		ListFn: func(_ context.Context) ([]domain.PlatformAdmin, error) {
			return want, nil
		},
	}, nil, nil)

	got, err := svc.ListPlatformAdmins(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// ListRBACAuditLog
// ============================================================================

func TestListRBACAuditLog_Success(t *testing.T) {
	want := []domain.RBACAuditEntry{
		{Action: domain.RBACActionGrant},
		{Action: domain.RBACActionRevoke},
	}
	svc := newService(nil, nil, nil, nil, &mockRBACAuditStore{
		ListFn: func(_ context.Context, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
			if limit != 50 || offset != 0 {
				t.Errorf("limit=%d offset=%d, want 50,0", limit, offset)
			}
			return want, 2, nil
		},
	}, nil)

	got, total, err := svc.ListRBACAuditLog(context.Background(), 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Errorf("total=%d len=%d, want 2,2", total, len(got))
	}
}

// ============================================================================
// BootstrapSuperAdmin
// ============================================================================

func TestBootstrapSuperAdmin_Success(t *testing.T) {
	auth0ID := "auth0|bootstrap"
	var paCreated, auditAppended bool

	svc := newService(&mockTxRunner{}, nil, nil, &mockPlatformAdminStore{
		CountByRoleFn: func(_ context.Context, role domain.PlatformAdminRole) (int, error) {
			if role != domain.PlatformAdminRoleSuperAdmin {
				t.Errorf("role = %v, want super_admin", role)
			}
			return 0, nil // no existing super admins
		},
		CreateFn: func(_ context.Context, pa *domain.PlatformAdmin) error {
			if pa.Role != domain.PlatformAdminRoleSuperAdmin {
				t.Errorf("created role = %v, want super_admin", pa.Role)
			}
			if pa.Auth0UserID != auth0ID {
				t.Errorf("auth0 user = %q, want %q", pa.Auth0UserID, auth0ID)
			}
			paCreated = true
			return nil
		},
	}, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, e *domain.RBACAuditEntry) error {
			if e.ActorAuth0UserID != "system:bootstrap" {
				t.Errorf("actor = %q, want system:bootstrap", e.ActorAuth0UserID)
			}
			if e.TargetAuth0UserID != auth0ID {
				t.Errorf("target = %q, want %q", e.TargetAuth0UserID, auth0ID)
			}
			if e.Action != domain.RBACActionGrant {
				t.Errorf("action = %v, want grant", e.Action)
			}
			auditAppended = true
			return nil
		},
	}, nil)

	err := svc.BootstrapSuperAdmin(context.Background(), auth0ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !paCreated {
		t.Fatal("PlatformAdmin.Create was not called")
	}
	if !auditAppended {
		t.Fatal("RBACAudit.Append was not called")
	}
}

func TestBootstrapSuperAdmin_AlreadyExists(t *testing.T) {
	var createCalled bool

	svc := newService(nil, nil, nil, &mockPlatformAdminStore{
		CountByRoleFn: func(_ context.Context, _ domain.PlatformAdminRole) (int, error) {
			return 1, nil // super admin already exists
		},
		CreateFn: func(_ context.Context, _ *domain.PlatformAdmin) error {
			createCalled = true
			return nil
		},
	}, nil, nil)

	err := svc.BootstrapSuperAdmin(context.Background(), "auth0|bootstrap")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Fatal("Create should not be called when super_admin already exists")
	}
}

func TestBootstrapSuperAdmin_EmptyAuth0ID(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)

	err := svc.BootstrapSuperAdmin(context.Background(), "")
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// CreateSeller
// ============================================================================

func TestCreateSeller_Success(t *testing.T) {
	userID := "auth0|user1"
	ctx := ctxWithUser(userID)

	var sellerCreated, ownerCreated bool
	svc := newService(&mockTxRunner{}, &mockSellerStore{
		GetBySlugFn: func(_ context.Context, _ string) (*domain.Seller, error) {
			return nil, nil // no slug conflict
		},
		CreateFn: func(_ context.Context, s *domain.Seller) error {
			if s.Status != domain.SellerStatusPending {
				t.Errorf("initial status = %v, want pending", s.Status)
			}
			sellerCreated = true
			return nil
		},
	}, &mockSellerUserStore{
		CreateFn: func(_ context.Context, su *domain.SellerUser) error {
			if su.Auth0UserID != userID {
				t.Errorf("owner auth0 = %q, want %q", su.Auth0UserID, userID)
			}
			if su.Role != domain.SellerUserRoleOwner {
				t.Errorf("owner role = %v, want owner", su.Role)
			}
			ownerCreated = true
			return nil
		},
	}, nil, nil, nil)

	err := svc.CreateSeller(ctx, &domain.Seller{Name: "Shop", Slug: "shop"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sellerCreated {
		t.Fatal("Seller.Create was not called")
	}
	if !ownerCreated {
		t.Fatal("SellerUser.Create (owner) was not called")
	}
}

func TestCreateSeller_SlugConflict(t *testing.T) {
	ctx := ctxWithUser("auth0|user1")

	svc := newService(nil, &mockSellerStore{
		GetBySlugFn: func(_ context.Context, _ string) (*domain.Seller, error) {
			return &domain.Seller{}, nil // slug already taken
		},
	}, nil, nil, nil, nil)

	err := svc.CreateSeller(ctx, &domain.Seller{Name: "Shop", Slug: "shop"})
	requireAppError(t, err, http.StatusConflict)
}

func TestCreateSeller_NoCallerIdentity(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)

	err := svc.CreateSeller(context.Background(), &domain.Seller{Name: "Shop", Slug: "shop"})
	requireAppError(t, err, http.StatusUnauthorized)
}

// ============================================================================
// AddSellerUser (RBAC)
// ============================================================================

func TestAddSellerUser_Success(t *testing.T) {
	sid := uuid.New()
	actorID := "auth0|owner"
	targetID := "auth0|new_member"
	ctx := ctxWithUser(actorID)

	var userCreated, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, auth0 string) (domain.SellerUserRole, error) {
			if auth0 == actorID {
				return domain.SellerUserRoleOwner, nil
			}
			return "", nil
		},
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.SellerUser, error) {
			return nil, nil // target doesn't exist yet
		},
		CreateFn: func(_ context.Context, su *domain.SellerUser) error {
			if su.Role != domain.SellerUserRoleMember {
				t.Errorf("role = %v, want member", su.Role)
			}
			userCreated = true
			return nil
		},
	}, nil, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, e *domain.RBACAuditEntry) error {
			if e.Action != domain.RBACActionGrant {
				t.Errorf("action = %v, want grant", e.Action)
			}
			auditLogged = true
			return nil
		},
	}, nil)

	created, err := svc.AddSellerUser(ctx, sid, targetID, domain.SellerUserRoleMember)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil {
		t.Fatal("expected non-nil created user")
	}
	if !userCreated {
		t.Fatal("SellerUser.Create was not called")
	}
	if !auditLogged {
		t.Fatal("RBACAudit.Append was not called")
	}
}

func TestAddSellerUser_OwnerRoleRejected(t *testing.T) {
	ctx := ctxWithUser("auth0|owner")
	svc := newService(nil, nil, nil, nil, nil, nil)

	_, err := svc.AddSellerUser(ctx, uuid.New(), "auth0|target", domain.SellerUserRoleOwner)
	requireAppError(t, err, http.StatusBadRequest)
}

func TestAddSellerUser_InvalidRole(t *testing.T) {
	ctx := ctxWithUser("auth0|owner")
	svc := newService(nil, nil, nil, nil, nil, nil)

	_, err := svc.AddSellerUser(ctx, uuid.New(), "auth0|target", domain.SellerUserRole("invalid"))
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// UpdateSellerUserRole (RBAC)
// ============================================================================

func TestUpdateSellerUserRole_Success(t *testing.T) {
	sid := uuid.New()
	targetID := uuid.New()
	actorID := "auth0|owner"
	ctx := ctxWithUser(actorID)

	var roleUpdated, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, auth0 string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          id,
				SellerID:    sid,
				Auth0UserID: "auth0|target",
				Role:        domain.SellerUserRoleMember,
			}, nil
		},
		UpdateRoleFn: func(_ context.Context, _ uuid.UUID, role domain.SellerUserRole) error {
			if role != domain.SellerUserRoleAdmin {
				t.Errorf("role = %v, want admin", role)
			}
			roleUpdated = true
			return nil
		},
	}, nil, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, e *domain.RBACAuditEntry) error {
			auditLogged = true
			return nil
		},
	}, nil)

	err := svc.UpdateSellerUserRole(ctx, sid, targetID, domain.SellerUserRoleAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !roleUpdated {
		t.Fatal("UpdateRole was not called")
	}
	if !auditLogged {
		t.Fatal("RBACAudit.Append was not called")
	}
}

func TestUpdateSellerUserRole_SelfRoleChange(t *testing.T) {
	sid := uuid.New()
	actorID := "auth0|owner"
	targetID := uuid.New()
	ctx := ctxWithUser(actorID)

	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          targetID,
				SellerID:    sid,
				Auth0UserID: actorID, // same user
				Role:        domain.SellerUserRoleOwner,
			}, nil
		},
	}, nil, nil, nil)

	err := svc.UpdateSellerUserRole(ctx, sid, targetID, domain.SellerUserRoleAdmin)
	requireAppError(t, err, http.StatusForbidden)
}

func TestUpdateSellerUserRole_LastOwnerDemotion(t *testing.T) {
	sid := uuid.New()
	actorID := "auth0|actor"
	targetID := uuid.New()
	ctx := ctxWithUser(actorID)

	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          targetID,
				SellerID:    sid,
				Auth0UserID: "auth0|target",
				Role:        domain.SellerUserRoleOwner,
			}, nil
		},
		CountByRoleFn: func(_ context.Context, _ uuid.UUID, _ domain.SellerUserRole) (int, error) {
			return 1, nil // only one owner
		},
	}, nil, nil, nil)

	err := svc.UpdateSellerUserRole(ctx, sid, targetID, domain.SellerUserRoleMember)
	requireAppError(t, err, http.StatusConflict)
}

// ============================================================================
// RemoveSellerUser (RBAC)
// ============================================================================

func TestRemoveSellerUser_Success(t *testing.T) {
	sid := uuid.New()
	targetID := uuid.New()
	actorID := "auth0|owner"
	ctx := ctxWithUser(actorID)

	var deleted, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          targetID,
				SellerID:    sid,
				Auth0UserID: "auth0|target",
				Role:        domain.SellerUserRoleMember,
			}, nil
		},
		DeleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}, nil, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, e *domain.RBACAuditEntry) error {
			if e.Action != domain.RBACActionRevoke {
				t.Errorf("action = %v, want revoke", e.Action)
			}
			auditLogged = true
			return nil
		},
	}, nil)

	err := svc.RemoveSellerUser(ctx, sid, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete was not called")
	}
	if !auditLogged {
		t.Fatal("RBACAudit.Append was not called")
	}
}

func TestRemoveSellerUser_LastOwner(t *testing.T) {
	sid := uuid.New()
	targetID := uuid.New()
	ctx := ctxWithUser("auth0|owner")

	svc := newService(&mockTxRunner{}, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
			GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerUser, error) {
				return &domain.SellerUser{
					ID:          targetID,
					SellerID:    sid,
					Auth0UserID: "auth0|other",
					Role:        domain.SellerUserRoleOwner,
				}, nil
			},
			CountByRoleFn: func(_ context.Context, _ uuid.UUID, _ domain.SellerUserRole) (int, error) {
				return 1, nil // last owner
			},
		},
		nil, nil, nil,
	)

	err := svc.RemoveSellerUser(ctx, sid, targetID)
	requireAppError(t, err, http.StatusConflict)
}

// ============================================================================
// GrantPlatformAdmin (RBAC)
// ============================================================================

func TestGrantPlatformAdmin_Success(t *testing.T) {
	actorID := "auth0|sa"
	targetID := "auth0|new_admin"
	ctx := ctxWithUser(actorID)

	var created, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, auth0 string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByAuth0IDFn: func(_ context.Context, _ string) (*domain.PlatformAdmin, error) {
			return nil, nil // target doesn't exist yet
		},
		CreateFn: func(_ context.Context, pa *domain.PlatformAdmin) error {
			if pa.Role != domain.PlatformAdminRoleAdmin {
				t.Errorf("role = %v, want admin", pa.Role)
			}
			created = true
			return nil
		},
	}, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, e *domain.RBACAuditEntry) error {
			auditLogged = true
			return nil
		},
	}, nil)

	pa, err := svc.GrantPlatformAdmin(ctx, targetID, domain.PlatformAdminRoleAdmin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pa == nil {
		t.Fatal("expected non-nil result")
	}
	if !created || !auditLogged {
		t.Fatal("expected Create and Append to be called")
	}
}

func TestGrantPlatformAdmin_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)
	_, err := svc.GrantPlatformAdmin(context.Background(),
		"auth0|target", domain.PlatformAdminRoleSupport)
	requireAppError(t, err, http.StatusUnauthorized)
}

// ============================================================================
// UpdatePlatformAdminRole (RBAC)
// ============================================================================

func TestUpdatePlatformAdminRole_SelfRoleChange(t *testing.T) {
	actorID := "auth0|sa"
	targetID := uuid.New()
	ctx := ctxWithUser(actorID)

	svc := newService(&mockTxRunner{}, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{
				ID:          targetID,
				Auth0UserID: actorID, // same user
				Role:        domain.PlatformAdminRoleSuperAdmin,
			}, nil
		},
	}, nil, nil)

	err := svc.UpdatePlatformAdminRole(ctx, targetID, domain.PlatformAdminRoleAdmin)
	requireAppError(t, err, http.StatusForbidden)
}

func TestUpdatePlatformAdminRole_LastSuperAdmin(t *testing.T) {
	actorID := "auth0|sa1"
	targetID := uuid.New()
	ctx := ctxWithUser(actorID)

	svc := newService(&mockTxRunner{}, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{
				ID:          targetID,
				Auth0UserID: "auth0|sa2", // different user
				Role:        domain.PlatformAdminRoleSuperAdmin,
			}, nil
		},
		CountByRoleFn: func(_ context.Context, _ domain.PlatformAdminRole) (int, error) {
			return 1, nil // last super admin
		},
	}, nil, nil)

	err := svc.UpdatePlatformAdminRole(ctx, targetID, domain.PlatformAdminRoleAdmin)
	requireAppError(t, err, http.StatusConflict)
}

func TestUpdatePlatformAdminRole_InvalidRole(t *testing.T) {
	targetID := uuid.New()
	ctx := ctxWithUser("auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
		},
		nil, nil,
	)

	err := svc.UpdatePlatformAdminRole(ctx, targetID, "bogus_role")
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// RevokePlatformAdmin (RBAC)
// ============================================================================

func TestRevokePlatformAdmin_Success(t *testing.T) {
	actorID := "auth0|sa"
	targetID := uuid.New()
	ctx := ctxWithUser(actorID)

	var deleted, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{
				ID:          targetID,
				Auth0UserID: "auth0|target",
				Role:        domain.PlatformAdminRoleAdmin,
			}, nil
		},
		DeleteFn: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, e *domain.RBACAuditEntry) error {
			if e.Action != domain.RBACActionRevoke {
				t.Errorf("action = %v, want revoke", e.Action)
			}
			auditLogged = true
			return nil
		},
	}, nil)

	err := svc.RevokePlatformAdmin(ctx, targetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("Delete was not called")
	}
	if !auditLogged {
		t.Fatal("RBACAudit.Append was not called")
	}
}

func TestRevokePlatformAdmin_LastSuperAdmin(t *testing.T) {
	targetID := uuid.New()
	ctx := ctxWithUser("auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
			GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PlatformAdmin, error) {
				return &domain.PlatformAdmin{
					ID: targetID, Auth0UserID: "auth0|other",
					Role: domain.PlatformAdminRoleSuperAdmin,
				}, nil
			},
			CountByRoleFn: func(_ context.Context, _ domain.PlatformAdminRole) (int, error) {
				return 1, nil // last super admin
			},
		},
		nil, nil,
	)

	err := svc.RevokePlatformAdmin(ctx, targetID)
	requireAppError(t, err, http.StatusConflict)
}

func TestRevokePlatformAdmin_Self(t *testing.T) {
	targetID := uuid.New()
	ctx := ctxWithUser("auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
			GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PlatformAdmin, error) {
				return &domain.PlatformAdmin{
					ID: targetID, Auth0UserID: "auth0|super", // same as actor
					Role: domain.PlatformAdminRoleSuperAdmin,
				}, nil
			},
		},
		nil, nil,
	)

	err := svc.RevokePlatformAdmin(ctx, targetID)
	requireAppError(t, err, http.StatusForbidden)
}

func TestRevokePlatformAdmin_TargetNotFound(t *testing.T) {
	ctx := ctxWithUser("auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
			GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PlatformAdmin, error) {
				return nil, nil
			},
		},
		nil, nil,
	)
	err := svc.RevokePlatformAdmin(ctx, uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestRevokePlatformAdmin_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)
	err := svc.RevokePlatformAdmin(context.Background(), uuid.New())
	requireAppError(t, err, http.StatusUnauthorized)
}

// ============================================================================
// TransferSellerOwnership (RBAC)
// ============================================================================

func TestTransferSellerOwnership_Success(t *testing.T) {
	sid := uuid.New()
	actorUserID := "auth0|owner"
	actorDBID := uuid.New()
	newOwnerID := uuid.New()
	ctx := ctxWithUser(actorUserID)

	var updateCalls int
	var auditCalls int
	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, auth0 string) (*domain.SellerUser, error) {
			if auth0 == actorUserID {
				return &domain.SellerUser{
					ID:          actorDBID,
					SellerID:    sid,
					Auth0UserID: actorUserID,
					Role:        domain.SellerUserRoleOwner,
				}, nil
			}
			return nil, nil
		},
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.SellerUser, error) {
			if id == newOwnerID {
				return &domain.SellerUser{
					ID:          newOwnerID,
					SellerID:    sid,
					Auth0UserID: "auth0|target",
					Role:        domain.SellerUserRoleAdmin,
				}, nil
			}
			return nil, nil
		},
		UpdateRoleFn: func(_ context.Context, _ uuid.UUID, _ domain.SellerUserRole) error {
			updateCalls++
			return nil
		},
	}, nil, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, _ *domain.RBACAuditEntry) error {
			auditCalls++
			return nil
		},
	}, nil)

	err := svc.TransferSellerOwnership(ctx, sid, newOwnerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateCalls != 2 {
		t.Errorf("UpdateRole called %d times, want 2 (demote actor + promote target)", updateCalls)
	}
	if auditCalls != 2 {
		t.Errorf("Append called %d times, want 2 (transfer + demotion)", auditCalls)
	}
}

func TestTransferSellerOwnership_NotOwner(t *testing.T) {
	sid := uuid.New()
	ctx := ctxWithUser("auth0|member")

	svc := newService(&mockTxRunner{}, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				Role: domain.SellerUserRoleMember, // not owner
			}, nil
		},
	}, nil, nil, nil)

	err := svc.TransferSellerOwnership(ctx, sid, uuid.New())
	requireAppError(t, err, http.StatusForbidden)
}

// ============================================================================
// IssueAPIToken
// ============================================================================

func TestIssueAPIToken_Success(t *testing.T) {
	sid := uuid.New()
	ctx := ctxWithUser("auth0|owner")

	var createCalled bool
	svc := newService(&mockTxRunner{}, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
		},
		nil, nil,
		&mockAPITokenStore{
			CreateFn: func(_ context.Context, tok *domain.SellerAPIToken) error {
				createCalled = true
				if tok.SellerID != sid {
					t.Errorf("sellerID mismatch: %s", tok.SellerID)
				}
				if tok.TokenPrefix != "sk_live_" {
					t.Errorf("prefix = %q, want sk_live_", tok.TokenPrefix)
				}
				if tok.TokenLookup == "" || len(tok.TokenHash) == 0 {
					t.Error("lookup or hash was empty")
				}
				return nil
			},
		},
	)

	tok, plaintext, err := svc.IssueAPIToken(ctx, sid, "My Token",
		[]domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil, "sk_live_")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == nil || plaintext == "" {
		t.Fatal("token or plaintext is empty")
	}
	if !createCalled {
		t.Fatal("Create was not called")
	}
}

func TestIssueAPIToken_ValidationErrors(t *testing.T) {
	sid := uuid.New()
	ctx := ctxWithUser("auth0|owner")
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-time.Hour)
	zero, neg := 0, -1

	cases := []struct {
		name       string
		in         func() (string, []domain.APITokenScope, *int, *int, *time.Time)
		wantStatus int
	}{
		{"empty name", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "", []domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil
		}, http.StatusBadRequest},
		{"blank name", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "   ", []domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil
		}, http.StatusBadRequest},
		{"too long name", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			long := make([]byte, 121)
			for i := range long {
				long[i] = 'a'
			}
			return string(long), []domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil
		}, http.StatusBadRequest},
		{"no scopes", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "n", nil, nil, nil, nil
		}, http.StatusBadRequest},
		{"invalid scope", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "n", []domain.APITokenScope{"bogus:scope"}, nil, nil, nil
		}, http.StatusBadRequest},
		{"expires in past", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "n", []domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, &past
		}, http.StatusBadRequest},
		{"zero rps", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "n", []domain.APITokenScope{domain.ScopeProductsRead}, &zero, nil, &future
		}, http.StatusBadRequest},
		{"neg burst", func() (string, []domain.APITokenScope, *int, *int, *time.Time) {
			return "n", []domain.APITokenScope{domain.ScopeProductsRead}, nil, &neg, &future
		}, http.StatusBadRequest},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := newService(&mockTxRunner{}, nil, nil, nil, nil, nil)
			name, scopes, rps, burst, exp := c.in()
			_, _, err := svc.IssueAPIToken(ctx, sid, name, scopes, rps, burst, exp, "sk_live_")
			requireAppError(t, err, c.wantStatus)
		})
	}
}

func TestIssueAPIToken_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)
	_, _, err := svc.IssueAPIToken(context.Background(), uuid.New(),
		"n", []domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil, "sk_live_")
	requireAppError(t, err, http.StatusUnauthorized)
}

func TestIssueAPIToken_InsufficientRole(t *testing.T) {
	sid := uuid.New()
	ctx := ctxWithUser("auth0|member")

	svc := newService(&mockTxRunner{}, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleMember, nil
			},
		},
		nil, nil, nil,
	)

	_, _, err := svc.IssueAPIToken(ctx, sid, "n",
		[]domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil, "sk_live_")
	requireAppError(t, err, http.StatusForbidden)
}

func TestIssueAPIToken_DeduplicatesScopes(t *testing.T) {
	sid := uuid.New()
	ctx := ctxWithUser("auth0|owner")

	var gotScopes []domain.APITokenScope
	svc := newService(&mockTxRunner{}, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
		},
		nil, nil,
		&mockAPITokenStore{
			CreateFn: func(_ context.Context, tok *domain.SellerAPIToken) error {
				gotScopes = tok.Scopes
				return nil
			},
		},
	)

	_, _, err := svc.IssueAPIToken(ctx, sid, "n",
		[]domain.APITokenScope{domain.ScopeProductsRead, domain.ScopeProductsRead, domain.ScopeOrdersRead},
		nil, nil, nil, "sk_live_")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotScopes) != 2 {
		t.Fatalf("len(scopes) = %d, want 2 after dedupe", len(gotScopes))
	}
}

// ============================================================================
// ListAPITokens / GetAPIToken
// ============================================================================

func TestListAPITokens_Success(t *testing.T) {
	sid := uuid.New()
	want := []domain.SellerAPIToken{{Name: "t1"}, {Name: "t2"}}
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		ListBySellerFn: func(_ context.Context, _ uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error) {
			if limit != 10 || offset != 0 {
				t.Errorf("limit=%d offset=%d, want 10,0", limit, offset)
			}
			return want, 2, nil
		},
	})

	got, total, err := svc.ListAPITokens(context.Background(), sid, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Errorf("total=%d len=%d, want 2,2", total, len(got))
	}
}

func TestListAPITokens_StoreError(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		ListBySellerFn: func(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.SellerAPIToken, int, error) {
			return nil, 0, errors.New("boom")
		},
	})
	_, _, err := svc.ListAPITokens(context.Background(), uuid.New(), 10, 0)
	requireAppError(t, err, http.StatusInternalServerError)
}

func TestGetAPIToken_Success(t *testing.T) {
	sid, id := uuid.New(), uuid.New()
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, qid uuid.UUID) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{ID: qid, SellerID: sid}, nil
		},
	})

	tok, err := svc.GetAPIToken(context.Background(), sid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.ID != id {
		t.Errorf("ID = %v, want %v", tok.ID, id)
	}
}

func TestGetAPIToken_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, nil
		},
	})
	_, err := svc.GetAPIToken(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestGetAPIToken_WrongSeller(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{SellerID: uuid.New()}, nil
		},
	})
	_, err := svc.GetAPIToken(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestGetAPIToken_StoreError(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, errors.New("boom")
		},
	})
	_, err := svc.GetAPIToken(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusInternalServerError)
}

// ============================================================================
// RevokeAPIToken
// ============================================================================

func TestRevokeAPIToken_Success(t *testing.T) {
	sid, id := uuid.New(), uuid.New()
	ctx := ctxWithUser("auth0|owner")

	var revoked bool
	svc := newService(&mockTxRunner{}, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
		},
		nil, nil,
		&mockAPITokenStore{
			GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
				return &domain.SellerAPIToken{
					ID: id, SellerID: sid,
					TokenPrefix: "sk_live_", TokenLookup: "abc123def456",
				}, nil
			},
			RevokeFn: func(_ context.Context, _ uuid.UUID, _ string) error {
				revoked = true
				return nil
			},
		},
	)

	prefix, lookup, err := svc.RevokeAPIToken(ctx, sid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !revoked {
		t.Fatal("Revoke was not called")
	}
	if prefix != "sk_live_" || lookup != "abc123def456" {
		t.Errorf("prefix/lookup = %q/%q, want sk_live_/abc123def456", prefix, lookup)
	}
}

func TestRevokeAPIToken_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)
	_, _, err := svc.RevokeAPIToken(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusUnauthorized)
}

func TestRevokeAPIToken_NotFound(t *testing.T) {
	ctx := ctxWithUser("auth0|owner")

	svc := newService(&mockTxRunner{}, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, nil
		},
	})
	_, _, err := svc.RevokeAPIToken(ctx, uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestRevokeAPIToken_WrongSeller(t *testing.T) {
	ctx := ctxWithUser("auth0|owner")

	svc := newService(&mockTxRunner{}, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{SellerID: uuid.New()}, nil
		},
	})
	_, _, err := svc.RevokeAPIToken(ctx, uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestRevokeAPIToken_InsufficientRole(t *testing.T) {
	sid := uuid.New()
	ctx := ctxWithUser("auth0|member")

	svc := newService(&mockTxRunner{}, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleMember, nil
			},
		},
		nil, nil,
		&mockAPITokenStore{
			GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
				return &domain.SellerAPIToken{SellerID: sid}, nil
			},
		},
	)

	_, _, err := svc.RevokeAPIToken(ctx, sid, uuid.New())
	requireAppError(t, err, http.StatusForbidden)
}

func TestRevokeAPIToken_StoreError(t *testing.T) {
	ctx := ctxWithUser("auth0|owner")

	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, errors.New("boom")
		},
	})
	_, _, err := svc.RevokeAPIToken(ctx, uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusInternalServerError)
}

// ============================================================================
// LookupAPIToken
// ============================================================================

// hashSecret returns sha256(secret) as []byte — matches the format stored
// in TokenHash so tests can build tokens with a known secret.
func hashSecret(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

func TestLookupAPIToken_Success(t *testing.T) {
	const secret = "my-test-secret"
	hash := hashSecret(secret)

	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, prefix, lookup string) (*domain.SellerAPIToken, error) {
			if prefix != "sk_live_" || lookup != "lkp123" {
				t.Errorf("prefix=%q lookup=%q", prefix, lookup)
			}
			return &domain.SellerAPIToken{TokenHash: hash, ID: uuid.New()}, nil
		},
	})

	tok, err := svc.LookupAPIToken(context.Background(), "sk_live_", "lkp123", secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == nil {
		t.Fatal("token is nil")
	}
}

func TestLookupAPIToken_InvalidFormat(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil)
	_, err := svc.LookupAPIToken(context.Background(), "", "x", "y")
	if !errors.Is(err, domain.ErrAPITokenInvalidFormat) {
		t.Errorf("err = %v, want ErrAPITokenInvalidFormat", err)
	}
}

func TestLookupAPIToken_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return nil, nil
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", "sec")
	if !errors.Is(err, domain.ErrAPITokenNotFound) {
		t.Errorf("err = %v, want ErrAPITokenNotFound", err)
	}
}

func TestLookupAPIToken_StoreError(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return nil, errors.New("boom")
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", "sec")
	requireAppError(t, err, http.StatusInternalServerError)
}

func TestLookupAPIToken_SecretMismatch(t *testing.T) {
	hash := hashSecret("correct-secret")
	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{TokenHash: hash}, nil
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", "wrongsecret")
	if !errors.Is(err, domain.ErrAPITokenNotFound) {
		t.Errorf("err = %v, want ErrAPITokenNotFound (secret mismatch is opaque)", err)
	}
}

func TestLookupAPIToken_Revoked(t *testing.T) {
	const secret = "sec"
	hash := hashSecret(secret)
	revokedAt := time.Now()

	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{TokenHash: hash, RevokedAt: &revokedAt}, nil
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", secret)
	if !errors.Is(err, domain.ErrAPITokenRevoked) {
		t.Errorf("err = %v, want ErrAPITokenRevoked", err)
	}
}

func TestLookupAPIToken_Expired(t *testing.T) {
	const secret = "sec"
	hash := hashSecret(secret)
	expired := time.Now().Add(-time.Hour)

	svc := newService(nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{TokenHash: hash, ExpiresAt: &expired}, nil
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", secret)
	if !errors.Is(err, domain.ErrAPITokenExpired) {
		t.Errorf("err = %v, want ErrAPITokenExpired", err)
	}
}
