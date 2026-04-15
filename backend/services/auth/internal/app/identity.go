package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/tenant"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
	"github.com/Riku-KANO/ec-test/services/auth/internal/port"
)

// IdentityService owns seller registration/approval.
// It intentionally does not reach into RBAC, subscription, or credential
// concerns; those live on sibling services that can be composed via
// AuthService (see auth_service.go) or wired individually.
type IdentityService struct {
	db          port.TxRunner
	sellers     port.SellerStore
	sellerUsers port.SellerUserStore
}

// NewIdentityService constructs an IdentityService. The sellerUsers store is
// required so CreateSeller can install the caller as the initial owner in
// the same transaction.
func NewIdentityService(
	db port.TxRunner,
	sellers port.SellerStore,
	sellerUsers port.SellerUserStore,
) *IdentityService {
	return &IdentityService{
		db:          db,
		sellers:     sellers,
		sellerUsers: sellerUsers,
	}
}

// CreateSeller creates a new seller and records the calling user as the
// initial owner of that seller organization. Both inserts happen in a
// single transaction so the seller is never left without an owner.
func (s *IdentityService) CreateSeller(ctx context.Context, seller *domain.Seller) error {
	// The caller becomes the initial owner. Without an identified caller we
	// cannot set up the owning user_role, so fail loudly.
	tc, err := tenant.FromContext(ctx)
	if err != nil || tc.UserID == "" {
		return apperrors.Unauthorized("caller identity required to create seller")
	}

	// Check slug uniqueness (non-TX; UNIQUE constraint is the
	// source of truth for concurrent races).
	existing, err := s.sellers.GetBySlug(ctx, seller.Slug)
	if err != nil {
		return apperrors.Internal("failed to check seller slug", err)
	}
	if existing != nil {
		return apperrors.Conflict("seller slug already exists")
	}

	seller.Status = domain.SellerStatusPending

	err = s.db.RunTx(ctx, func(txCtx context.Context) error {
		if err := s.sellers.Create(txCtx, seller); err != nil {
			return err
		}
		owner := &domain.SellerUser{
			SellerID:    seller.ID,
			Auth0UserID: tc.UserID,
			Role:        domain.SellerUserRoleOwner,
		}
		return s.sellerUsers.Create(txCtx, owner)
	})
	if err != nil {
		return apperrors.Internal("failed to create seller", err)
	}

	slog.Info("seller created", "id", seller.ID, "slug", seller.Slug, "owner", tc.UserID)
	return nil
}

// GetSeller retrieves a seller by ID.
func (s *IdentityService) GetSeller(ctx context.Context, id uuid.UUID) (*domain.Seller, error) {
	seller, err := s.sellers.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return nil, apperrors.NotFound("seller not found")
	}
	return seller, nil
}

// ListSellers returns a paginated list of sellers.
func (s *IdentityService) ListSellers(ctx context.Context, limit, offset int) ([]domain.Seller, int, error) {
	sellers, total, err := s.sellers.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list sellers", err)
	}
	return sellers, total, nil
}

// BatchGetSellers resolves a batch of seller ids. Unknown ids are silently
// omitted from the result so callers must tolerate length mismatch. Used by
// the /internal/sellers/batch-get endpoint for service-to-service snapshotting.
func (s *IdentityService) BatchGetSellers(ctx context.Context, ids []uuid.UUID) ([]domain.Seller, error) {
	sellers, err := s.sellers.BatchGetByIDs(ctx, ids)
	if err != nil {
		return nil, apperrors.Internal("failed to batch get sellers", err)
	}
	return sellers, nil
}

// ApproveSeller transitions a seller to the approved status.
func (s *IdentityService) ApproveSeller(ctx context.Context, id uuid.UUID) error {
	seller, err := s.sellers.GetByID(ctx, id)
	if err != nil {
		return apperrors.Internal("failed to get seller", err)
	}
	if seller == nil {
		return apperrors.NotFound("seller not found")
	}
	if seller.Status != domain.SellerStatusPending {
		return apperrors.BadRequest("seller is not in pending status")
	}

	if err := s.sellers.UpdateStatus(ctx, id, domain.SellerStatusApproved); err != nil {
		return apperrors.Internal("failed to approve seller", err)
	}

	slog.Info("seller approved", "id", id)
	return nil
}
