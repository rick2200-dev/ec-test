package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/repository"
)

// AuthService implements business logic for auth operations.
type AuthService struct {
	tenants *repository.TenantRepository
	sellers *repository.SellerRepository
}

// NewAuthService creates a new AuthService.
func NewAuthService(tenants *repository.TenantRepository, sellers *repository.SellerRepository) *AuthService {
	return &AuthService{
		tenants: tenants,
		sellers: sellers,
	}
}

// CreateTenant creates a new tenant.
func (s *AuthService) CreateTenant(ctx context.Context, t *domain.Tenant) error {
	// Check for slug uniqueness.
	existing, err := s.tenants.GetBySlug(ctx, t.Slug)
	if err != nil {
		return apperrors.Internal("failed to check tenant slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("tenant slug already exists")
	}

	t.Status = domain.TenantStatusActive
	if err := s.tenants.Create(ctx, t); err != nil {
		return apperrors.Internal("failed to create tenant", err)
	}

	slog.Info("tenant created", "id", t.ID, "slug", t.Slug)
	return nil
}

// GetTenant retrieves a tenant by ID.
func (s *AuthService) GetTenant(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, err := s.tenants.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get tenant", err)
	}
	if t == nil {
		return nil, apperrors.NotFound("tenant not found")
	}
	return t, nil
}

// ListTenants returns a paginated list of tenants.
func (s *AuthService) ListTenants(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error) {
	tenants, total, err := s.tenants.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list tenants", err)
	}
	return tenants, total, nil
}

// CreateSeller creates a new seller within a tenant.
func (s *AuthService) CreateSeller(ctx context.Context, tenantID uuid.UUID, seller *domain.Seller) error {
	// Verify tenant exists.
	t, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return apperrors.Internal("failed to verify tenant", err)
	}
	if t == nil {
		return apperrors.NotFound("tenant not found")
	}

	// Check slug uniqueness within tenant.
	existing, err := s.sellers.GetBySlug(ctx, tenantID, seller.Slug)
	if err != nil {
		return apperrors.Internal("failed to check seller slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("seller slug already exists in this tenant")
	}

	seller.Status = domain.SellerStatusPending
	if err := s.sellers.Create(ctx, tenantID, seller); err != nil {
		return apperrors.Internal("failed to create seller", err)
	}

	slog.Info("seller created", "id", seller.ID, "tenant_id", tenantID, "slug", seller.Slug)
	return nil
}

// GetSeller retrieves a seller by ID within a tenant.
func (s *AuthService) GetSeller(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error) {
	seller, err := s.sellers.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return nil, apperrors.NotFound("seller not found")
	}
	return seller, nil
}

// ListSellers returns a paginated list of sellers for a tenant.
func (s *AuthService) ListSellers(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error) {
	sellers, total, err := s.sellers.List(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list sellers", err)
	}
	return sellers, total, nil
}

// ApproveSeller transitions a seller to the approved status.
func (s *AuthService) ApproveSeller(ctx context.Context, tenantID, id uuid.UUID) error {
	seller, err := s.sellers.GetByID(ctx, tenantID, id)
	if err != nil {
		return apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return apperrors.NotFound("seller not found")
	}
	if seller.Status != domain.SellerStatusPending {
		return apperrors.BadRequest("seller is not in pending status")
	}

	if err := s.sellers.UpdateStatus(ctx, tenantID, id, domain.SellerStatusApproved); err != nil {
		return apperrors.Internal("failed to approve seller", err)
	}

	slog.Info("seller approved", "id", id, "tenant_id", tenantID)
	return nil
}
