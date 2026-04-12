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
	RunTenantTxFn func(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}

func (m *mockTxRunner) RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error {
	if m.RunTenantTxFn != nil {
		return m.RunTenantTxFn(ctx, tenantID, fn)
	}
	return fn(ctx)
}

type mockTenantStore struct {
	CreateFn  func(ctx context.Context, t *domain.Tenant) error
	GetByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlugFn func(ctx context.Context, slug string) (*domain.Tenant, error)
	ListFn    func(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error)
}

func (m *mockTenantStore) Create(ctx context.Context, t *domain.Tenant) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, t)
	}
	return nil
}
func (m *mockTenantStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockTenantStore) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	if m.GetBySlugFn != nil {
		return m.GetBySlugFn(ctx, slug)
	}
	return nil, nil
}
func (m *mockTenantStore) List(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

type mockSellerStore struct {
	GetByIDFn     func(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error)
	GetBySlugFn   func(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Seller, error)
	ListFn        func(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error)
	UpdateStatusFn func(ctx context.Context, tenantID, id uuid.UUID, status domain.SellerStatus) error
	CreateFn      func(ctx context.Context, tenantID uuid.UUID, s *domain.Seller) error
}

func (m *mockSellerStore) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSellerStore) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Seller, error) {
	if m.GetBySlugFn != nil {
		return m.GetBySlugFn(ctx, tenantID, slug)
	}
	return nil, nil
}
func (m *mockSellerStore) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}
func (m *mockSellerStore) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.SellerStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, tenantID, id, status)
	}
	return nil
}
func (m *mockSellerStore) Create(ctx context.Context, tenantID uuid.UUID, s *domain.Seller) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, tenantID, s)
	}
	return nil
}

type mockSellerUserStore struct {
	GetByIDFn     func(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerUser, error)
	GetByAuth0IDFn func(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error)
	ListBySellerFn func(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error)
	CreateFn      func(ctx context.Context, su *domain.SellerUser) error
	UpdateRoleFn  func(ctx context.Context, tenantID, id uuid.UUID, role domain.SellerUserRole) error
	DeleteFn      func(ctx context.Context, tenantID, id uuid.UUID) error
	CountByRoleFn func(ctx context.Context, tenantID, sellerID uuid.UUID, role domain.SellerUserRole) (int, error)
	CheckRoleFn   func(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error)
}

func (m *mockSellerUserStore) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerUser, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSellerUserStore) GetByAuth0ID(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error) {
	if m.GetByAuth0IDFn != nil {
		return m.GetByAuth0IDFn(ctx, tenantID, sellerID, auth0UserID)
	}
	return nil, nil
}
func (m *mockSellerUserStore) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error) {
	if m.ListBySellerFn != nil {
		return m.ListBySellerFn(ctx, tenantID, sellerID)
	}
	return nil, nil
}
func (m *mockSellerUserStore) Create(ctx context.Context, su *domain.SellerUser) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, su)
	}
	return nil
}
func (m *mockSellerUserStore) UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.SellerUserRole) error {
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, tenantID, id, role)
	}
	return nil
}
func (m *mockSellerUserStore) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, tenantID, id)
	}
	return nil
}
func (m *mockSellerUserStore) CountByRole(ctx context.Context, tenantID, sellerID uuid.UUID, role domain.SellerUserRole) (int, error) {
	if m.CountByRoleFn != nil {
		return m.CountByRoleFn(ctx, tenantID, sellerID, role)
	}
	return 0, nil
}
func (m *mockSellerUserStore) CheckRole(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (domain.SellerUserRole, error) {
	if m.CheckRoleFn != nil {
		return m.CheckRoleFn(ctx, tenantID, sellerID, auth0UserID)
	}
	return "", nil
}

type mockPlatformAdminStore struct {
	GetByIDFn     func(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformAdmin, error)
	GetByAuth0IDFn func(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error)
	ListFn        func(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error)
	CountByRoleFn func(ctx context.Context, tenantID uuid.UUID, role domain.PlatformAdminRole) (int, error)
	CreateFn      func(ctx context.Context, pa *domain.PlatformAdmin) error
	UpdateRoleFn  func(ctx context.Context, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error
	DeleteFn      func(ctx context.Context, tenantID, id uuid.UUID) error
	CheckRoleFn   func(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error)
}

func (m *mockPlatformAdminStore) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformAdmin, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockPlatformAdminStore) GetByAuth0ID(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error) {
	if m.GetByAuth0IDFn != nil {
		return m.GetByAuth0IDFn(ctx, tenantID, auth0UserID)
	}
	return nil, nil
}
func (m *mockPlatformAdminStore) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockPlatformAdminStore) CountByRole(ctx context.Context, tenantID uuid.UUID, role domain.PlatformAdminRole) (int, error) {
	if m.CountByRoleFn != nil {
		return m.CountByRoleFn(ctx, tenantID, role)
	}
	return 0, nil
}
func (m *mockPlatformAdminStore) Create(ctx context.Context, pa *domain.PlatformAdmin) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, pa)
	}
	return nil
}
func (m *mockPlatformAdminStore) UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error {
	if m.UpdateRoleFn != nil {
		return m.UpdateRoleFn(ctx, tenantID, id, role)
	}
	return nil
}
func (m *mockPlatformAdminStore) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, tenantID, id)
	}
	return nil
}
func (m *mockPlatformAdminStore) CheckRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error) {
	if m.CheckRoleFn != nil {
		return m.CheckRoleFn(ctx, tenantID, auth0UserID)
	}
	return "", nil
}

type mockRBACAuditStore struct {
	AppendFn      func(ctx context.Context, e *domain.RBACAuditEntry) error
	ListByTenantFn func(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error)
}

func (m *mockRBACAuditStore) Append(ctx context.Context, e *domain.RBACAuditEntry) error {
	if m.AppendFn != nil {
		return m.AppendFn(ctx, e)
	}
	return nil
}
func (m *mockRBACAuditStore) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
	if m.ListByTenantFn != nil {
		return m.ListByTenantFn(ctx, tenantID, limit, offset)
	}
	return nil, 0, nil
}

type mockSubscriptionStore struct {
	CreatePlanFn              func(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error
	GetPlanByIDFn             func(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionPlan, error)
	ListPlansFn               func(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error)
	UpdatePlanFn              func(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error
	GetSellerSubscriptionFn   func(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error)
	UpsertSellerSubscriptionFn func(ctx context.Context, tenantID uuid.UUID, sub *domain.SellerSubscription) error
	RefreshPlanBoostViewFn    func(ctx context.Context) error
}

func (m *mockSubscriptionStore) CreatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error {
	if m.CreatePlanFn != nil {
		return m.CreatePlanFn(ctx, tenantID, p)
	}
	return nil
}
func (m *mockSubscriptionStore) GetPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	if m.GetPlanByIDFn != nil {
		return m.GetPlanByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSubscriptionStore) ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error) {
	if m.ListPlansFn != nil {
		return m.ListPlansFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockSubscriptionStore) UpdatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error {
	if m.UpdatePlanFn != nil {
		return m.UpdatePlanFn(ctx, tenantID, p)
	}
	return nil
}
func (m *mockSubscriptionStore) GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
	if m.GetSellerSubscriptionFn != nil {
		return m.GetSellerSubscriptionFn(ctx, tenantID, sellerID)
	}
	return nil, nil
}
func (m *mockSubscriptionStore) UpsertSellerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.SellerSubscription) error {
	if m.UpsertSellerSubscriptionFn != nil {
		return m.UpsertSellerSubscriptionFn(ctx, tenantID, sub)
	}
	return nil
}
func (m *mockSubscriptionStore) RefreshPlanBoostView(ctx context.Context) error {
	if m.RefreshPlanBoostViewFn != nil {
		return m.RefreshPlanBoostViewFn(ctx)
	}
	return nil
}

type mockBuyerSubscriptionStore struct {
	CreateBuyerPlanFn         func(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error
	GetBuyerPlanByIDFn        func(ctx context.Context, tenantID, id uuid.UUID) (*domain.BuyerPlan, error)
	ListBuyerPlansFn          func(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error)
	UpdateBuyerPlanFn         func(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error
	GetBuyerSubscriptionFn    func(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error)
	UpsertBuyerSubscriptionFn func(ctx context.Context, tenantID uuid.UUID, sub *domain.BuyerSubscription) error
}

func (m *mockBuyerSubscriptionStore) CreateBuyerPlan(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error {
	if m.CreateBuyerPlanFn != nil {
		return m.CreateBuyerPlanFn(ctx, tenantID, p)
	}
	return nil
}
func (m *mockBuyerSubscriptionStore) GetBuyerPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.BuyerPlan, error) {
	if m.GetBuyerPlanByIDFn != nil {
		return m.GetBuyerPlanByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockBuyerSubscriptionStore) ListBuyerPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.BuyerPlan, error) {
	if m.ListBuyerPlansFn != nil {
		return m.ListBuyerPlansFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockBuyerSubscriptionStore) UpdateBuyerPlan(ctx context.Context, tenantID uuid.UUID, p *domain.BuyerPlan) error {
	if m.UpdateBuyerPlanFn != nil {
		return m.UpdateBuyerPlanFn(ctx, tenantID, p)
	}
	return nil
}
func (m *mockBuyerSubscriptionStore) GetBuyerSubscription(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error) {
	if m.GetBuyerSubscriptionFn != nil {
		return m.GetBuyerSubscriptionFn(ctx, tenantID, buyerAuth0ID)
	}
	return nil, nil
}
func (m *mockBuyerSubscriptionStore) UpsertBuyerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.BuyerSubscription) error {
	if m.UpsertBuyerSubscriptionFn != nil {
		return m.UpsertBuyerSubscriptionFn(ctx, tenantID, sub)
	}
	return nil
}

type mockAPITokenStore struct {
	CreateFn        func(ctx context.Context, t *domain.SellerAPIToken) error
	GetByIDFn       func(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerAPIToken, error)
	ListBySellerFn  func(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error)
	RevokeFn        func(ctx context.Context, tenantID, id uuid.UUID, actorAuth0UserID string) error
	GetByLookupFn   func(ctx context.Context, prefix, lookup string) (*domain.SellerAPIToken, error)
	TouchLastUsedAtFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockAPITokenStore) Create(ctx context.Context, t *domain.SellerAPIToken) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, t)
	}
	return nil
}
func (m *mockAPITokenStore) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerAPIToken, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockAPITokenStore) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error) {
	if m.ListBySellerFn != nil {
		return m.ListBySellerFn(ctx, tenantID, sellerID, limit, offset)
	}
	return nil, 0, nil
}
func (m *mockAPITokenStore) Revoke(ctx context.Context, tenantID, id uuid.UUID, actorAuth0UserID string) error {
	if m.RevokeFn != nil {
		return m.RevokeFn(ctx, tenantID, id, actorAuth0UserID)
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
	tenants *mockTenantStore,
	sellers *mockSellerStore,
	sellerUsers *mockSellerUserStore,
	platformAdmins *mockPlatformAdminStore,
	rbacAudit *mockRBACAuditStore,
	subscriptions *mockSubscriptionStore,
	buyerSubscriptions *mockBuyerSubscriptionStore,
	apiTokens *mockAPITokenStore,
) *app.AuthService {
	if db == nil {
		db = &mockTxRunner{}
	}
	if tenants == nil {
		tenants = &mockTenantStore{}
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
	if subscriptions == nil {
		subscriptions = &mockSubscriptionStore{}
	}
	if buyerSubscriptions == nil {
		buyerSubscriptions = &mockBuyerSubscriptionStore{}
	}
	if apiTokens == nil {
		apiTokens = &mockAPITokenStore{}
	}
	return app.NewAuthService(db, tenants, sellers, sellerUsers, platformAdmins, rbacAudit, subscriptions, buyerSubscriptions, apiTokens)
}

// requireAppError asserts the error is an *apperrors.AppError with the expected
// HTTP status code and returns it.
func requireAppError(t *testing.T, err error, wantStatus int) *apperrors.AppError {
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
	return appErr
}

// ctxWithUser returns a context enriched with tenant context for the given
// tenant ID and Auth0 user ID.
func ctxWithUser(tenantID uuid.UUID, userID string) context.Context {
	return tenant.WithContext(context.Background(), tenant.Context{
		TenantID: tenantID,
		UserID:   userID,
	})
}

// ============================================================================
// 1. CreateTenant
// ============================================================================

func TestCreateTenant_Success(t *testing.T) {
	var created bool
	svc := newService(nil, &mockTenantStore{
		GetBySlugFn: func(_ context.Context, slug string) (*domain.Tenant, error) {
			return nil, nil // no conflict
		},
		CreateFn: func(_ context.Context, t *domain.Tenant) error {
			created = true
			if t.Status != domain.TenantStatusActive {
				return errors.New("expected status to be set to active")
			}
			return nil
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	err := svc.CreateTenant(context.Background(), &domain.Tenant{Name: "Test", Slug: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("Create was not called")
	}
}

func TestCreateTenant_SlugConflict(t *testing.T) {
	svc := newService(nil, &mockTenantStore{
		GetBySlugFn: func(_ context.Context, slug string) (*domain.Tenant, error) {
			return &domain.Tenant{Slug: slug}, nil // existing tenant
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	err := svc.CreateTenant(context.Background(), &domain.Tenant{Name: "Test", Slug: "test"})
	requireAppError(t, err, http.StatusConflict)
}

// ============================================================================
// 2. GetTenant
// ============================================================================

func TestGetTenant_Success(t *testing.T) {
	id := uuid.New()
	svc := newService(nil, &mockTenantStore{
		GetByIDFn: func(_ context.Context, qid uuid.UUID) (*domain.Tenant, error) {
			if qid != id {
				t.Errorf("queried id = %v, want %v", qid, id)
			}
			return &domain.Tenant{ID: id, Name: "T"}, nil
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	got, err := svc.GetTenant(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID = %v, want %v", got.ID, id)
	}
}

func TestGetTenant_NotFound(t *testing.T) {
	svc := newService(nil, &mockTenantStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetTenant(context.Background(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 3. ListTenants
// ============================================================================

func TestListTenants_Success(t *testing.T) {
	want := []domain.Tenant{{Name: "A"}, {Name: "B"}}
	svc := newService(nil, &mockTenantStore{
		ListFn: func(_ context.Context, limit, offset int) ([]domain.Tenant, int, error) {
			if limit != 10 || offset != 0 {
				t.Errorf("limit=%d offset=%d, want 10,0", limit, offset)
			}
			return want, 2, nil
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	got, total, err := svc.ListTenants(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// 4. GetSeller
// ============================================================================

func TestGetSeller_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, tenantID, id uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{ID: sid, TenantID: tenantID}, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	got, err := svc.GetSeller(context.Background(), tid, sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != sid {
		t.Errorf("ID = %v, want %v", got.ID, sid)
	}
}

func TestGetSeller_NotFound(t *testing.T) {
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	_, err := svc.GetSeller(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 5. ListSellers
// ============================================================================

func TestListSellers_Success(t *testing.T) {
	tid := uuid.New()
	want := []domain.Seller{{Name: "S1"}, {Name: "S2"}}
	svc := newService(nil, nil, &mockSellerStore{
		ListFn: func(_ context.Context, _ uuid.UUID, limit, offset int) ([]domain.Seller, int, error) {
			return want, 2, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	got, total, err := svc.ListSellers(context.Background(), tid, 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Errorf("total=%d len=%d, want 2,2", total, len(got))
	}
}

// ============================================================================
// 6. ApproveSeller
// ============================================================================

func TestApproveSeller_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	var statusUpdated bool
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{ID: sid, Status: domain.SellerStatusPending}, nil
		},
		UpdateStatusFn: func(_ context.Context, _, _ uuid.UUID, status domain.SellerStatus) error {
			if status != domain.SellerStatusApproved {
				t.Errorf("status = %v, want approved", status)
			}
			statusUpdated = true
			return nil
		},
	}, nil, nil, nil, nil, nil, nil)

	err := svc.ApproveSeller(context.Background(), tid, sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !statusUpdated {
		t.Fatal("UpdateStatus was not called")
	}
}

func TestApproveSeller_NotFound(t *testing.T) {
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	err := svc.ApproveSeller(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestApproveSeller_NotPending(t *testing.T) {
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{Status: domain.SellerStatusApproved}, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	err := svc.ApproveSeller(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// 7. CreatePlan
// ============================================================================

func TestCreatePlan_Success(t *testing.T) {
	tid := uuid.New()
	var created bool
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		CreatePlanFn: func(_ context.Context, _ uuid.UUID, p *domain.SubscriptionPlan) error {
			if p.Status != "active" {
				t.Errorf("default status = %q, want active", p.Status)
			}
			if p.PriceCurrency != "JPY" {
				t.Errorf("default currency = %q, want JPY", p.PriceCurrency)
			}
			created = true
			return nil
		},
	}, nil, nil)

	plan := &domain.SubscriptionPlan{Name: "Pro", Slug: "pro"}
	err := svc.CreatePlan(context.Background(), tid, plan)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("CreatePlan was not called")
	}
}

// ============================================================================
// 8. GetPlan
// ============================================================================

func TestGetPlan_Success(t *testing.T) {
	tid := uuid.New()
	pid := uuid.New()
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		GetPlanByIDFn: func(_ context.Context, _, id uuid.UUID) (*domain.SubscriptionPlan, error) {
			return &domain.SubscriptionPlan{ID: id, Name: "Pro"}, nil
		},
	}, nil, nil)

	got, err := svc.GetPlan(context.Background(), tid, pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != pid {
		t.Errorf("ID = %v, want %v", got.ID, pid)
	}
}

func TestGetPlan_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		GetPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionPlan, error) {
			return nil, nil
		},
	}, nil, nil)

	_, err := svc.GetPlan(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 9. ListPlans
// ============================================================================

func TestListPlans_Success(t *testing.T) {
	tid := uuid.New()
	want := []domain.SubscriptionPlan{{Name: "Free"}, {Name: "Pro"}}
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		ListPlansFn: func(_ context.Context, _ uuid.UUID) ([]domain.SubscriptionPlan, error) {
			return want, nil
		},
	}, nil, nil)

	got, err := svc.ListPlans(context.Background(), tid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// 10. UpdatePlan
// ============================================================================

func TestUpdatePlan_Success(t *testing.T) {
	tid := uuid.New()
	pid := uuid.New()
	var updated bool
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		GetPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionPlan, error) {
			return &domain.SubscriptionPlan{ID: pid}, nil
		},
		UpdatePlanFn: func(_ context.Context, _ uuid.UUID, p *domain.SubscriptionPlan) error {
			updated = true
			return nil
		},
	}, nil, nil)

	err := svc.UpdatePlan(context.Background(), tid, &domain.SubscriptionPlan{ID: pid, Name: "Pro+"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("UpdatePlan was not called")
	}
}

func TestUpdatePlan_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		GetPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionPlan, error) {
			return nil, nil
		},
	}, nil, nil)

	err := svc.UpdatePlan(context.Background(), uuid.New(), &domain.SubscriptionPlan{ID: uuid.New()})
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 11. GetSellerSubscription
// ============================================================================

func TestGetSellerSubscription_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		GetSellerSubscriptionFn: func(_ context.Context, _, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
			return &domain.SellerSubscriptionWithPlan{
				SellerSubscription: domain.SellerSubscription{SellerID: sellerID},
				PlanName:           "Pro",
			}, nil
		},
	}, nil, nil)

	got, err := svc.GetSellerSubscription(context.Background(), tid, sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SellerID != sid {
		t.Errorf("SellerID = %v, want %v", got.SellerID, sid)
	}
}

func TestGetSellerSubscription_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, &mockSubscriptionStore{
		GetSellerSubscriptionFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
			return nil, nil
		},
	}, nil, nil)

	_, err := svc.GetSellerSubscription(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 12. SubscribeSeller
// ============================================================================

func TestSubscribeSeller_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	pid := uuid.New()
	var upserted bool

	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{ID: sid}, nil
		},
	}, nil, nil, nil, &mockSubscriptionStore{
		GetPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionPlan, error) {
			return &domain.SubscriptionPlan{ID: pid}, nil
		},
		UpsertSellerSubscriptionFn: func(_ context.Context, _ uuid.UUID, sub *domain.SellerSubscription) error {
			if sub.Status != domain.SubscriptionStatusActive {
				t.Errorf("status = %v, want active", sub.Status)
			}
			if sub.PlanID != pid {
				t.Errorf("planID = %v, want %v", sub.PlanID, pid)
			}
			upserted = true
			return nil
		},
		RefreshPlanBoostViewFn: func(_ context.Context) error { return nil },
	}, nil, nil)

	sub, err := svc.SubscribeSeller(context.Background(), tid, sid, pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upserted {
		t.Fatal("UpsertSellerSubscription was not called")
	}
	if sub.PlanID != pid {
		t.Errorf("returned sub.PlanID = %v, want %v", sub.PlanID, pid)
	}
}

func TestSubscribeSeller_SellerNotFound(t *testing.T) {
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil, nil)

	_, err := svc.SubscribeSeller(context.Background(), uuid.New(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestSubscribeSeller_PlanNotFound(t *testing.T) {
	svc := newService(nil, nil, &mockSellerStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Seller, error) {
			return &domain.Seller{}, nil
		},
	}, nil, nil, nil, &mockSubscriptionStore{
		GetPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionPlan, error) {
			return nil, nil
		},
	}, nil, nil)

	_, err := svc.SubscribeSeller(context.Background(), uuid.New(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 13. CreateBuyerPlan
// ============================================================================

func TestCreateBuyerPlan_Success(t *testing.T) {
	tid := uuid.New()
	var created bool
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		CreateBuyerPlanFn: func(_ context.Context, _ uuid.UUID, p *domain.BuyerPlan) error {
			if p.Status != "active" {
				t.Errorf("default status = %q, want active", p.Status)
			}
			if p.PriceCurrency != "JPY" {
				t.Errorf("default currency = %q, want JPY", p.PriceCurrency)
			}
			created = true
			return nil
		},
	}, nil)

	err := svc.CreateBuyerPlan(context.Background(), tid, &domain.BuyerPlan{Name: "Basic"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatal("CreateBuyerPlan was not called")
	}
}

// ============================================================================
// 14. GetBuyerPlan
// ============================================================================

func TestGetBuyerPlan_Success(t *testing.T) {
	tid := uuid.New()
	pid := uuid.New()
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerPlanByIDFn: func(_ context.Context, _, id uuid.UUID) (*domain.BuyerPlan, error) {
			return &domain.BuyerPlan{ID: id, Name: "Basic"}, nil
		},
	}, nil)

	got, err := svc.GetBuyerPlan(context.Background(), tid, pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != pid {
		t.Errorf("ID = %v, want %v", got.ID, pid)
	}
}

func TestGetBuyerPlan_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.BuyerPlan, error) {
			return nil, nil
		},
	}, nil)

	_, err := svc.GetBuyerPlan(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 15. ListBuyerPlans
// ============================================================================

func TestListBuyerPlans_Success(t *testing.T) {
	tid := uuid.New()
	want := []domain.BuyerPlan{{Name: "Basic"}, {Name: "Premium"}}
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		ListBuyerPlansFn: func(_ context.Context, _ uuid.UUID) ([]domain.BuyerPlan, error) {
			return want, nil
		},
	}, nil)

	got, err := svc.ListBuyerPlans(context.Background(), tid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// 16. UpdateBuyerPlan
// ============================================================================

func TestUpdateBuyerPlan_Success(t *testing.T) {
	tid := uuid.New()
	pid := uuid.New()
	var updated bool
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.BuyerPlan, error) {
			return &domain.BuyerPlan{ID: pid}, nil
		},
		UpdateBuyerPlanFn: func(_ context.Context, _ uuid.UUID, p *domain.BuyerPlan) error {
			updated = true
			return nil
		},
	}, nil)

	err := svc.UpdateBuyerPlan(context.Background(), tid, &domain.BuyerPlan{ID: pid, Name: "Premium+"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated {
		t.Fatal("UpdateBuyerPlan was not called")
	}
}

func TestUpdateBuyerPlan_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.BuyerPlan, error) {
			return nil, nil
		},
	}, nil)

	err := svc.UpdateBuyerPlan(context.Background(), uuid.New(), &domain.BuyerPlan{ID: uuid.New()})
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 17. GetBuyerSubscription
// ============================================================================

func TestGetBuyerSubscription_Success(t *testing.T) {
	tid := uuid.New()
	buyerID := "auth0|buyer1"
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerSubscriptionFn: func(_ context.Context, _ uuid.UUID, auth0ID string) (*domain.BuyerSubscriptionWithPlan, error) {
			return &domain.BuyerSubscriptionWithPlan{
				BuyerSubscription: domain.BuyerSubscription{BuyerAuth0ID: auth0ID},
				PlanName:          "Basic",
			}, nil
		},
	}, nil)

	got, err := svc.GetBuyerSubscription(context.Background(), tid, buyerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.BuyerAuth0ID != buyerID {
		t.Errorf("BuyerAuth0ID = %q, want %q", got.BuyerAuth0ID, buyerID)
	}
}

func TestGetBuyerSubscription_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerSubscriptionFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.BuyerSubscriptionWithPlan, error) {
			return nil, nil
		},
	}, nil)

	_, err := svc.GetBuyerSubscription(context.Background(), uuid.New(), "auth0|nobody")
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 18. SubscribeBuyer
// ============================================================================

func TestSubscribeBuyer_Success(t *testing.T) {
	tid := uuid.New()
	pid := uuid.New()
	buyerID := "auth0|buyer1"
	var upserted bool

	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.BuyerPlan, error) {
			return &domain.BuyerPlan{ID: pid}, nil
		},
		UpsertBuyerSubscriptionFn: func(_ context.Context, _ uuid.UUID, sub *domain.BuyerSubscription) error {
			if sub.Status != domain.SubscriptionStatusActive {
				t.Errorf("status = %v, want active", sub.Status)
			}
			if sub.BuyerAuth0ID != buyerID {
				t.Errorf("BuyerAuth0ID = %q, want %q", sub.BuyerAuth0ID, buyerID)
			}
			upserted = true
			return nil
		},
	}, nil)

	sub, err := svc.SubscribeBuyer(context.Background(), tid, buyerID, pid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upserted {
		t.Fatal("UpsertBuyerSubscription was not called")
	}
	if sub.PlanID != pid {
		t.Errorf("returned sub.PlanID = %v, want %v", sub.PlanID, pid)
	}
}

func TestSubscribeBuyer_PlanNotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, &mockBuyerSubscriptionStore{
		GetBuyerPlanByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.BuyerPlan, error) {
			return nil, nil
		},
	}, nil)

	_, err := svc.SubscribeBuyer(context.Background(), uuid.New(), "auth0|buyer", uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

// ============================================================================
// 19. LookupSellerRole
// ============================================================================

func TestLookupSellerRole_Found(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	svc := newService(nil, nil, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _, _ uuid.UUID, auth0UserID string) (*domain.SellerUser, error) {
			return &domain.SellerUser{Role: domain.SellerUserRoleOwner, Auth0UserID: auth0UserID}, nil
		},
	}, nil, nil, nil, nil, nil)

	role, err := svc.LookupSellerRole(context.Background(), tid, sid, "auth0|user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != domain.SellerUserRoleOwner {
		t.Errorf("role = %q, want %q", role, domain.SellerUserRoleOwner)
	}
}

func TestLookupSellerRole_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _, _ uuid.UUID, _ string) (*domain.SellerUser, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil)

	role, err := svc.LookupSellerRole(context.Background(), uuid.New(), uuid.New(), "auth0|nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != "" {
		t.Errorf("role = %q, want empty string", role)
	}
}

// ============================================================================
// 20. LookupPlatformAdminRole
// ============================================================================

func TestLookupPlatformAdminRole_Found(t *testing.T) {
	tid := uuid.New()
	svc := newService(nil, nil, nil, nil, &mockPlatformAdminStore{
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{Role: domain.PlatformAdminRoleSuperAdmin}, nil
		},
	}, nil, nil, nil, nil)

	role, err := svc.LookupPlatformAdminRole(context.Background(), tid, "auth0|admin1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != domain.PlatformAdminRoleSuperAdmin {
		t.Errorf("role = %q, want %q", role, domain.PlatformAdminRoleSuperAdmin)
	}
}

func TestLookupPlatformAdminRole_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, &mockPlatformAdminStore{
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.PlatformAdmin, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil)

	role, err := svc.LookupPlatformAdminRole(context.Background(), uuid.New(), "auth0|nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role != "" {
		t.Errorf("role = %q, want empty string", role)
	}
}

// ============================================================================
// 21. ListSellerTeam
// ============================================================================

func TestListSellerTeam_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	want := []domain.SellerUser{
		{Auth0UserID: "auth0|u1", Role: domain.SellerUserRoleOwner},
		{Auth0UserID: "auth0|u2", Role: domain.SellerUserRoleMember},
	}
	svc := newService(nil, nil, nil, &mockSellerUserStore{
		ListBySellerFn: func(_ context.Context, _, _ uuid.UUID) ([]domain.SellerUser, error) {
			return want, nil
		},
	}, nil, nil, nil, nil, nil)

	got, err := svc.ListSellerTeam(context.Background(), tid, sid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// 22. ListPlatformAdmins
// ============================================================================

func TestListPlatformAdmins_Success(t *testing.T) {
	tid := uuid.New()
	want := []domain.PlatformAdmin{
		{Auth0UserID: "auth0|sa1", Role: domain.PlatformAdminRoleSuperAdmin},
		{Auth0UserID: "auth0|a1", Role: domain.PlatformAdminRoleAdmin},
	}
	svc := newService(nil, nil, nil, nil, &mockPlatformAdminStore{
		ListFn: func(_ context.Context, _ uuid.UUID) ([]domain.PlatformAdmin, error) {
			return want, nil
		},
	}, nil, nil, nil, nil)

	got, err := svc.ListPlatformAdmins(context.Background(), tid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

// ============================================================================
// 23. ListRBACAuditLog
// ============================================================================

func TestListRBACAuditLog_Success(t *testing.T) {
	tid := uuid.New()
	want := []domain.RBACAuditEntry{
		{Action: domain.RBACActionGrant},
		{Action: domain.RBACActionRevoke},
	}
	svc := newService(nil, nil, nil, nil, nil, &mockRBACAuditStore{
		ListByTenantFn: func(_ context.Context, _ uuid.UUID, limit, offset int) ([]domain.RBACAuditEntry, int, error) {
			if limit != 50 || offset != 0 {
				t.Errorf("limit=%d offset=%d, want 50,0", limit, offset)
			}
			return want, 2, nil
		},
	}, nil, nil, nil)

	got, total, err := svc.ListRBACAuditLog(context.Background(), tid, 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Errorf("total=%d len=%d, want 2,2", total, len(got))
	}
}

// ============================================================================
// 24. BootstrapSuperAdmin
// ============================================================================

func TestBootstrapSuperAdmin_Success(t *testing.T) {
	tid := uuid.New()
	auth0ID := "auth0|bootstrap"
	var paCreated, auditAppended bool

	svc := newService(&mockTxRunner{}, nil, nil, nil, &mockPlatformAdminStore{
		CountByRoleFn: func(_ context.Context, _ uuid.UUID, role domain.PlatformAdminRole) (int, error) {
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
	}, nil, nil, nil)

	err := svc.BootstrapSuperAdmin(context.Background(), tid, auth0ID)
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
	tid := uuid.New()
	var createCalled bool

	svc := newService(nil, nil, nil, nil, &mockPlatformAdminStore{
		CountByRoleFn: func(_ context.Context, _ uuid.UUID, _ domain.PlatformAdminRole) (int, error) {
			return 1, nil // super admin already exists
		},
		CreateFn: func(_ context.Context, _ *domain.PlatformAdmin) error {
			createCalled = true
			return nil
		},
	}, nil, nil, nil, nil)

	err := svc.BootstrapSuperAdmin(context.Background(), tid, "auth0|bootstrap")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Fatal("Create should not be called when super_admin already exists")
	}
}

func TestBootstrapSuperAdmin_EmptyAuth0ID(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	err := svc.BootstrapSuperAdmin(context.Background(), uuid.New(), "")
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// CreateSeller (covers tenant context requirement)
// ============================================================================

func TestCreateSeller_Success(t *testing.T) {
	tid := uuid.New()
	userID := "auth0|user1"
	ctx := ctxWithUser(tid, userID)

	var sellerCreated, ownerCreated bool
	svc := newService(&mockTxRunner{}, &mockTenantStore{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
			return &domain.Tenant{ID: id}, nil
		},
	}, &mockSellerStore{
		GetBySlugFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.Seller, error) {
			return nil, nil // no slug conflict
		},
		CreateFn: func(_ context.Context, _ uuid.UUID, s *domain.Seller) error {
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
	}, nil, nil, nil, nil, nil)

	err := svc.CreateSeller(ctx, tid, &domain.Seller{Name: "Shop", Slug: "shop"})
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

func TestCreateSeller_TenantNotFound(t *testing.T) {
	ctx := ctxWithUser(uuid.New(), "auth0|user1")

	svc := newService(nil, &mockTenantStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
			return nil, nil
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	err := svc.CreateSeller(ctx, uuid.New(), &domain.Seller{Name: "Shop", Slug: "shop"})
	requireAppError(t, err, http.StatusNotFound)
}

func TestCreateSeller_SlugConflict(t *testing.T) {
	tid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|user1")

	svc := newService(nil, &mockTenantStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
			return &domain.Tenant{ID: tid}, nil
		},
	}, &mockSellerStore{
		GetBySlugFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.Seller, error) {
			return &domain.Seller{}, nil // slug already taken
		},
	}, nil, nil, nil, nil, nil, nil)

	err := svc.CreateSeller(ctx, tid, &domain.Seller{Name: "Shop", Slug: "shop"})
	requireAppError(t, err, http.StatusConflict)
}

func TestCreateSeller_NoCallerIdentity(t *testing.T) {
	tid := uuid.New()
	// Use plain context without tenant.WithContext
	svc := newService(nil, &mockTenantStore{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
			return &domain.Tenant{ID: tid}, nil
		},
	}, nil, nil, nil, nil, nil, nil, nil)

	err := svc.CreateSeller(context.Background(), tid, &domain.Seller{Name: "Shop", Slug: "shop"})
	requireAppError(t, err, http.StatusUnauthorized)
}

// ============================================================================
// AddSellerUser (RBAC)
// ============================================================================

func TestAddSellerUser_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	actorID := "auth0|owner"
	targetID := "auth0|new_member"
	ctx := ctxWithUser(tid, actorID)

	var userCreated, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, auth0 string) (domain.SellerUserRole, error) {
			if auth0 == actorID {
				return domain.SellerUserRoleOwner, nil
			}
			return "", nil
		},
		GetByAuth0IDFn: func(_ context.Context, _, _ uuid.UUID, _ string) (*domain.SellerUser, error) {
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
	}, nil, nil, nil)

	created, err := svc.AddSellerUser(ctx, tid, sid, targetID, domain.SellerUserRoleMember)
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
	ctx := ctxWithUser(uuid.New(), "auth0|owner")
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.AddSellerUser(ctx, uuid.New(), uuid.New(), "auth0|target", domain.SellerUserRoleOwner)
	requireAppError(t, err, http.StatusBadRequest)
}

func TestAddSellerUser_InvalidRole(t *testing.T) {
	ctx := ctxWithUser(uuid.New(), "auth0|owner")
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	_, err := svc.AddSellerUser(ctx, uuid.New(), uuid.New(), "auth0|target", domain.SellerUserRole("invalid"))
	requireAppError(t, err, http.StatusBadRequest)
}

// ============================================================================
// UpdateSellerUserRole (RBAC)
// ============================================================================

func TestUpdateSellerUserRole_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	targetID := uuid.New()
	actorID := "auth0|owner"
	ctx := ctxWithUser(tid, actorID)

	var roleUpdated, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, auth0 string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _, id uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          id,
				SellerID:    sid,
				Auth0UserID: "auth0|target",
				Role:        domain.SellerUserRoleMember,
			}, nil
		},
		UpdateRoleFn: func(_ context.Context, _, _ uuid.UUID, role domain.SellerUserRole) error {
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
	}, nil, nil, nil)

	err := svc.UpdateSellerUserRole(ctx, tid, sid, targetID, domain.SellerUserRoleAdmin)
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
	tid := uuid.New()
	sid := uuid.New()
	actorID := "auth0|owner"
	targetID := uuid.New()
	ctx := ctxWithUser(tid, actorID)

	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          targetID,
				SellerID:    sid,
				Auth0UserID: actorID, // same user
				Role:        domain.SellerUserRoleOwner,
			}, nil
		},
	}, nil, nil, nil, nil, nil)

	err := svc.UpdateSellerUserRole(ctx, tid, sid, targetID, domain.SellerUserRoleAdmin)
	requireAppError(t, err, http.StatusForbidden)
}

func TestUpdateSellerUserRole_LastOwnerDemotion(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	actorID := "auth0|actor"
	targetID := uuid.New()
	ctx := ctxWithUser(tid, actorID)

	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          targetID,
				SellerID:    sid,
				Auth0UserID: "auth0|target",
				Role:        domain.SellerUserRoleOwner,
			}, nil
		},
		CountByRoleFn: func(_ context.Context, _, _ uuid.UUID, _ domain.SellerUserRole) (int, error) {
			return 1, nil // only one owner
		},
	}, nil, nil, nil, nil, nil)

	err := svc.UpdateSellerUserRole(ctx, tid, sid, targetID, domain.SellerUserRoleMember)
	requireAppError(t, err, http.StatusConflict)
}

// ============================================================================
// RemoveSellerUser (RBAC)
// ============================================================================

func TestRemoveSellerUser_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	targetID := uuid.New()
	actorID := "auth0|owner"
	ctx := ctxWithUser(tid, actorID)

	var deleted, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
			return domain.SellerUserRoleOwner, nil
		},
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				ID:          targetID,
				SellerID:    sid,
				Auth0UserID: "auth0|target",
				Role:        domain.SellerUserRoleMember,
			}, nil
		},
		DeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
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
	}, nil, nil, nil)

	err := svc.RemoveSellerUser(ctx, tid, sid, targetID)
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

// ============================================================================
// GrantPlatformAdmin (RBAC)
// ============================================================================

func TestGrantPlatformAdmin_Success(t *testing.T) {
	tid := uuid.New()
	actorID := "auth0|sa"
	targetID := "auth0|new_admin"
	ctx := ctxWithUser(tid, actorID)

	var created, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, auth0 string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByAuth0IDFn: func(_ context.Context, _ uuid.UUID, _ string) (*domain.PlatformAdmin, error) {
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
	}, nil, nil, nil)

	pa, err := svc.GrantPlatformAdmin(ctx, tid, targetID, domain.PlatformAdminRoleAdmin)
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

// ============================================================================
// UpdatePlatformAdminRole (RBAC)
// ============================================================================

func TestUpdatePlatformAdminRole_SelfRoleChange(t *testing.T) {
	tid := uuid.New()
	actorID := "auth0|sa"
	targetID := uuid.New()
	ctx := ctxWithUser(tid, actorID)

	svc := newService(&mockTxRunner{}, nil, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{
				ID:          targetID,
				Auth0UserID: actorID, // same user
				Role:        domain.PlatformAdminRoleSuperAdmin,
			}, nil
		},
	}, nil, nil, nil, nil)

	err := svc.UpdatePlatformAdminRole(ctx, tid, targetID, domain.PlatformAdminRoleAdmin)
	requireAppError(t, err, http.StatusForbidden)
}

func TestUpdatePlatformAdminRole_LastSuperAdmin(t *testing.T) {
	tid := uuid.New()
	actorID := "auth0|sa1"
	targetID := uuid.New()
	ctx := ctxWithUser(tid, actorID)

	svc := newService(&mockTxRunner{}, nil, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{
				ID:          targetID,
				Auth0UserID: "auth0|sa2", // different user
				Role:        domain.PlatformAdminRoleSuperAdmin,
			}, nil
		},
		CountByRoleFn: func(_ context.Context, _ uuid.UUID, _ domain.PlatformAdminRole) (int, error) {
			return 1, nil // last super admin
		},
	}, nil, nil, nil, nil)

	err := svc.UpdatePlatformAdminRole(ctx, tid, targetID, domain.PlatformAdminRoleAdmin)
	requireAppError(t, err, http.StatusConflict)
}

// ============================================================================
// RevokePlatformAdmin (RBAC)
// ============================================================================

func TestRevokePlatformAdmin_Success(t *testing.T) {
	tid := uuid.New()
	actorID := "auth0|sa"
	targetID := uuid.New()
	ctx := ctxWithUser(tid, actorID)

	var deleted, auditLogged bool
	svc := newService(&mockTxRunner{}, nil, nil, nil, &mockPlatformAdminStore{
		CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
			return domain.PlatformAdminRoleSuperAdmin, nil
		},
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PlatformAdmin, error) {
			return &domain.PlatformAdmin{
				ID:          targetID,
				Auth0UserID: "auth0|target",
				Role:        domain.PlatformAdminRoleAdmin,
			}, nil
		},
		DeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
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
	}, nil, nil, nil)

	err := svc.RevokePlatformAdmin(ctx, tid, targetID)
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

// ============================================================================
// TransferSellerOwnership (RBAC)
// ============================================================================

func TestTransferSellerOwnership_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	actorUserID := "auth0|owner"
	actorDBID := uuid.New()
	newOwnerID := uuid.New()
	ctx := ctxWithUser(tid, actorUserID)

	var updateCalls int
	var auditCalls int
	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _, _ uuid.UUID, auth0 string) (*domain.SellerUser, error) {
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
		GetByIDFn: func(_ context.Context, _, id uuid.UUID) (*domain.SellerUser, error) {
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
		UpdateRoleFn: func(_ context.Context, _, _ uuid.UUID, _ domain.SellerUserRole) error {
			updateCalls++
			return nil
		},
	}, nil, &mockRBACAuditStore{
		AppendFn: func(_ context.Context, _ *domain.RBACAuditEntry) error {
			auditCalls++
			return nil
		},
	}, nil, nil, nil)

	err := svc.TransferSellerOwnership(ctx, tid, sid, newOwnerID)
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
	tid := uuid.New()
	sid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|member")

	svc := newService(&mockTxRunner{}, nil, nil, &mockSellerUserStore{
		GetByAuth0IDFn: func(_ context.Context, _, _ uuid.UUID, _ string) (*domain.SellerUser, error) {
			return &domain.SellerUser{
				Role: domain.SellerUserRoleMember, // not owner
			}, nil
		},
	}, nil, nil, nil, nil, nil)

	err := svc.TransferSellerOwnership(ctx, tid, sid, uuid.New())
	requireAppError(t, err, http.StatusForbidden)
}

// ============================================================================
// IssueAPIToken
// ============================================================================

func TestIssueAPIToken_Success(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	var createCalled bool
	svc := newService(&mockTxRunner{}, nil, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
		},
		nil, nil, nil, nil,
		&mockAPITokenStore{
			CreateFn: func(_ context.Context, tok *domain.SellerAPIToken) error {
				createCalled = true
				if tok.TenantID != tid || tok.SellerID != sid {
					t.Errorf("ids mismatch: tenantID=%s sellerID=%s", tok.TenantID, tok.SellerID)
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

	tok, plaintext, err := svc.IssueAPIToken(ctx, tid, sid, "My Token",
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
	tid := uuid.New()
	sid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-time.Hour)
	zero, neg := 0, -1

	cases := []struct {
		name    string
		in      func() (string, []domain.APITokenScope, *int, *int, *time.Time)
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
			svc := newService(&mockTxRunner{}, nil, nil, nil, nil, nil, nil, nil, nil)
			name, scopes, rps, burst, exp := c.in()
			_, _, err := svc.IssueAPIToken(ctx, tid, sid, name, scopes, rps, burst, exp, "sk_live_")
			requireAppError(t, err, c.wantStatus)
		})
	}
}

func TestIssueAPIToken_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, _, err := svc.IssueAPIToken(context.Background(), uuid.New(), uuid.New(),
		"n", []domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil, "sk_live_")
	requireAppError(t, err, http.StatusUnauthorized)
}

func TestIssueAPIToken_InsufficientRole(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|member")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleMember, nil
			},
		},
		nil, nil, nil, nil, nil,
	)

	_, _, err := svc.IssueAPIToken(ctx, tid, sid, "n",
		[]domain.APITokenScope{domain.ScopeProductsRead}, nil, nil, nil, "sk_live_")
	requireAppError(t, err, http.StatusForbidden)
}

func TestIssueAPIToken_DeduplicatesScopes(t *testing.T) {
	tid := uuid.New()
	sid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	var gotScopes []domain.APITokenScope
	svc := newService(&mockTxRunner{}, nil, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
		},
		nil, nil, nil, nil,
		&mockAPITokenStore{
			CreateFn: func(_ context.Context, tok *domain.SellerAPIToken) error {
				gotScopes = tok.Scopes
				return nil
			},
		},
	)

	_, _, err := svc.IssueAPIToken(ctx, tid, sid, "n",
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
	tid := uuid.New()
	sid := uuid.New()
	want := []domain.SellerAPIToken{{Name: "t1"}, {Name: "t2"}}
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		ListBySellerFn: func(_ context.Context, _, _ uuid.UUID, limit, offset int) ([]domain.SellerAPIToken, int, error) {
			if limit != 10 || offset != 0 {
				t.Errorf("limit=%d offset=%d, want 10,0", limit, offset)
			}
			return want, 2, nil
		},
	})

	got, total, err := svc.ListAPITokens(context.Background(), tid, sid, 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 || len(got) != 2 {
		t.Errorf("total=%d len=%d, want 2,2", total, len(got))
	}
}

func TestListAPITokens_StoreError(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		ListBySellerFn: func(_ context.Context, _, _ uuid.UUID, _, _ int) ([]domain.SellerAPIToken, int, error) {
			return nil, 0, errors.New("boom")
		},
	})
	_, _, err := svc.ListAPITokens(context.Background(), uuid.New(), uuid.New(), 10, 0)
	requireAppError(t, err, http.StatusInternalServerError)
}

func TestGetAPIToken_Success(t *testing.T) {
	tid, sid, id := uuid.New(), uuid.New(), uuid.New()
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, qid uuid.UUID) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{ID: qid, SellerID: sid}, nil
		},
	})

	tok, err := svc.GetAPIToken(context.Background(), tid, sid, id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.ID != id {
		t.Errorf("ID = %v, want %v", tok.ID, id)
	}
}

func TestGetAPIToken_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, nil
		},
	})
	_, err := svc.GetAPIToken(context.Background(), uuid.New(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestGetAPIToken_WrongSeller(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{SellerID: uuid.New()}, nil
		},
	})
	_, err := svc.GetAPIToken(context.Background(), uuid.New(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestGetAPIToken_StoreError(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, errors.New("boom")
		},
	})
	_, err := svc.GetAPIToken(context.Background(), uuid.New(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusInternalServerError)
}

// ============================================================================
// RevokeAPIToken
// ============================================================================

func TestRevokeAPIToken_Success(t *testing.T) {
	tid, sid, id := uuid.New(), uuid.New(), uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	var revoked bool
	svc := newService(&mockTxRunner{}, nil, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
		},
		nil, nil, nil, nil,
		&mockAPITokenStore{
			GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
				return &domain.SellerAPIToken{
					ID: id, SellerID: sid,
					TokenPrefix: "sk_live_", TokenLookup: "abc123def456",
				}, nil
			},
			RevokeFn: func(_ context.Context, _, _ uuid.UUID, _ string) error {
				revoked = true
				return nil
			},
		},
	)

	prefix, lookup, err := svc.RevokeAPIToken(ctx, tid, sid, id)
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
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, _, err := svc.RevokeAPIToken(context.Background(), uuid.New(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusUnauthorized)
}

func TestRevokeAPIToken_NotFound(t *testing.T) {
	tid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	svc := newService(&mockTxRunner{}, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, nil
		},
	})
	_, _, err := svc.RevokeAPIToken(ctx, tid, uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestRevokeAPIToken_WrongSeller(t *testing.T) {
	tid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	svc := newService(&mockTxRunner{}, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{SellerID: uuid.New()}, nil
		},
	})
	_, _, err := svc.RevokeAPIToken(ctx, tid, uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestRevokeAPIToken_InsufficientRole(t *testing.T) {
	tid, sid := uuid.New(), uuid.New()
	ctx := ctxWithUser(tid, "auth0|member")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleMember, nil
			},
		},
		nil, nil, nil, nil,
		&mockAPITokenStore{
			GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
				return &domain.SellerAPIToken{SellerID: sid}, nil
			},
		},
	)

	_, _, err := svc.RevokeAPIToken(ctx, tid, sid, uuid.New())
	requireAppError(t, err, http.StatusForbidden)
}

func TestRevokeAPIToken_StoreError(t *testing.T) {
	tid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerAPIToken, error) {
			return nil, errors.New("boom")
		},
	})
	_, _, err := svc.RevokeAPIToken(ctx, tid, uuid.New(), uuid.New())
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

	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
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
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.LookupAPIToken(context.Background(), "", "x", "y")
	if !errors.Is(err, domain.ErrAPITokenInvalidFormat) {
		t.Errorf("err = %v, want ErrAPITokenInvalidFormat", err)
	}
}

func TestLookupAPIToken_NotFound(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
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
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return nil, errors.New("boom")
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", "sec")
	requireAppError(t, err, http.StatusInternalServerError)
}

func TestLookupAPIToken_SecretMismatch(t *testing.T) {
	hash := hashSecret("correct-secret")
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
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

	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
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

	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, &mockAPITokenStore{
		GetByLookupFn: func(_ context.Context, _, _ string) (*domain.SellerAPIToken, error) {
			return &domain.SellerAPIToken{TokenHash: hash, ExpiresAt: &expired}, nil
		},
	})
	_, err := svc.LookupAPIToken(context.Background(), "sk_live_", "abc", secret)
	if !errors.Is(err, domain.ErrAPITokenExpired) {
		t.Errorf("err = %v, want ErrAPITokenExpired", err)
	}
}

// ============================================================================
// RBAC error-path coverage (pushes overall % over 80 by exercising sentinel
// mapping branches in mapRBACError and a few RBAC method failure modes).
// ============================================================================

func TestRemoveSellerUser_LastOwner(t *testing.T) {
	tid, sid, targetID := uuid.New(), uuid.New(), uuid.New()
	ctx := ctxWithUser(tid, "auth0|owner")

	svc := newService(&mockTxRunner{}, nil, nil,
		&mockSellerUserStore{
			CheckRoleFn: func(_ context.Context, _, _ uuid.UUID, _ string) (domain.SellerUserRole, error) {
				return domain.SellerUserRoleOwner, nil
			},
			GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SellerUser, error) {
				return &domain.SellerUser{
					ID: targetID, TenantID: tid, SellerID: sid,
					Auth0UserID: "auth0|other", Role: domain.SellerUserRoleOwner,
				}, nil
			},
			CountByRoleFn: func(_ context.Context, _, _ uuid.UUID, _ domain.SellerUserRole) (int, error) {
				return 1, nil // last owner
			},
		},
		nil, nil, nil, nil, nil,
	)

	err := svc.RemoveSellerUser(ctx, tid, sid, targetID)
	requireAppError(t, err, http.StatusConflict)
}

func TestRevokePlatformAdmin_LastSuperAdmin(t *testing.T) {
	tid, targetID := uuid.New(), uuid.New()
	ctx := ctxWithUser(tid, "auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
			GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PlatformAdmin, error) {
				return &domain.PlatformAdmin{
					ID: targetID, Auth0UserID: "auth0|other",
					Role: domain.PlatformAdminRoleSuperAdmin,
				}, nil
			},
			CountByRoleFn: func(_ context.Context, _ uuid.UUID, _ domain.PlatformAdminRole) (int, error) {
				return 1, nil // last super admin
			},
		},
		nil, nil, nil, nil,
	)

	err := svc.RevokePlatformAdmin(ctx, tid, targetID)
	requireAppError(t, err, http.StatusConflict)
}

func TestRevokePlatformAdmin_Self(t *testing.T) {
	tid, targetID := uuid.New(), uuid.New()
	ctx := ctxWithUser(tid, "auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
			GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PlatformAdmin, error) {
				return &domain.PlatformAdmin{
					ID: targetID, Auth0UserID: "auth0|super", // same as actor
					Role: domain.PlatformAdminRoleSuperAdmin,
				}, nil
			},
		},
		nil, nil, nil, nil,
	)

	err := svc.RevokePlatformAdmin(ctx, tid, targetID)
	requireAppError(t, err, http.StatusForbidden)
}

func TestRevokePlatformAdmin_TargetNotFound(t *testing.T) {
	tid := uuid.New()
	ctx := ctxWithUser(tid, "auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
			GetByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PlatformAdmin, error) {
				return nil, nil
			},
		},
		nil, nil, nil, nil,
	)
	err := svc.RevokePlatformAdmin(ctx, tid, uuid.New())
	requireAppError(t, err, http.StatusNotFound)
}

func TestRevokePlatformAdmin_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	err := svc.RevokePlatformAdmin(context.Background(), uuid.New(), uuid.New())
	requireAppError(t, err, http.StatusUnauthorized)
}

func TestUpdatePlatformAdminRole_InvalidRole(t *testing.T) {
	tid, targetID := uuid.New(), uuid.New()
	ctx := ctxWithUser(tid, "auth0|super")

	svc := newService(&mockTxRunner{}, nil, nil, nil,
		&mockPlatformAdminStore{
			CheckRoleFn: func(_ context.Context, _ uuid.UUID, _ string) (domain.PlatformAdminRole, error) {
				return domain.PlatformAdminRoleSuperAdmin, nil
			},
		},
		nil, nil, nil, nil,
	)

	err := svc.UpdatePlatformAdminRole(ctx, tid, targetID, "bogus_role")
	requireAppError(t, err, http.StatusBadRequest)
}

func TestGrantPlatformAdmin_Unauthorized(t *testing.T) {
	svc := newService(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	_, err := svc.GrantPlatformAdmin(context.Background(), uuid.New(),
		"auth0|target", domain.PlatformAdminRoleSupport)
	requireAppError(t, err, http.StatusUnauthorized)
}
