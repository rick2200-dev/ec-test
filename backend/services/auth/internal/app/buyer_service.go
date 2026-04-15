package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// BuyerRepo is the driven port for buyer profile persistence. Declared at
// the use-case side so callers can swap the implementation in tests.
type BuyerRepo interface {
	// Upsert creates the buyer row on first login and updates email /
	// display_name on subsequent logins. Keyed by auth0_sub.
	Upsert(ctx context.Context, b *domain.Buyer) error
}

// BuyerService is a thin use-case for buyer profile upsert. It is
// intentionally separate from AuthService because buyer identity is owned
// by Auth0 and these records exist only for profile-display convenience.
type BuyerService struct {
	repo BuyerRepo
}

// NewBuyerService constructs a BuyerService.
func NewBuyerService(repo BuyerRepo) *BuyerService {
	return &BuyerService{repo: repo}
}

// UpsertBuyer stores (or refreshes) the profile record for the Auth0
// subject. Validation is minimal — the BFF has already authenticated the
// user against Auth0 before this is called.
func (s *BuyerService) UpsertBuyer(
	ctx context.Context,
	auth0Sub, email, displayName string,
) (*domain.Buyer, error) {
	auth0Sub = strings.TrimSpace(auth0Sub)
	email = strings.TrimSpace(email)
	displayName = strings.TrimSpace(displayName)

	if auth0Sub == "" {
		return nil, fmt.Errorf("auth0_sub is required")
	}
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	b := &domain.Buyer{
		Auth0Sub:    auth0Sub,
		Email:       email,
		DisplayName: displayName,
	}
	if err := s.repo.Upsert(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}
