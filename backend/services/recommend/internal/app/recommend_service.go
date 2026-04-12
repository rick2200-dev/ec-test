package app

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/engine"
)

// ViewRefresher refreshes the materialized view for popular products.
// Declared here (consumer side) so tests can inject a fake without a real
// database. *repository.PostgresViewRefresher satisfies this interface.
type ViewRefresher interface {
	RefreshPopularProducts(ctx context.Context) error
}

// RecommendService provides business logic for product recommendations.
type RecommendService struct {
	engine    engine.RecommendEngine
	refresher ViewRefresher
}

// NewRecommendService creates a new RecommendService.
func NewRecommendService(eng engine.RecommendEngine, refresher ViewRefresher) *RecommendService {
	return &RecommendService{engine: eng, refresher: refresher}
}

// GetRecommendations validates the request and delegates to the engine.
func (s *RecommendService) GetRecommendations(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	if req.TenantID == uuid.Nil {
		return nil, domain.ErrMissingTenantID
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	switch req.Type {
	case domain.Popular, domain.Similar, domain.PersonalizedForYou, domain.FrequentlyBoughtTogether:
		// valid
	default:
		return nil, domain.ErrInvalidRecommendationType
	}

	if (req.Type == domain.Similar || req.Type == domain.FrequentlyBoughtTogether) && req.ProductID == nil {
		return nil, domain.ErrMissingProductID
	}

	resp, err := s.engine.Recommend(ctx, req)
	if err != nil {
		slog.Error("recommendation engine error", "type", req.Type, "error", err)
		return nil, apperrors.Internal("failed to get recommendations", err)
	}

	return resp, nil
}

// RecordUserEvent validates and records a user behavior event.
func (s *RecommendService) RecordUserEvent(ctx context.Context, event domain.UserEvent) error {
	if event.TenantID == uuid.Nil {
		return domain.ErrMissingTenantID
	}
	if event.UserID == "" {
		return domain.ErrMissingUserID
	}
	if event.ProductID == uuid.Nil {
		return domain.ErrMissingProductID
	}

	switch event.EventType {
	case domain.ProductViewed, domain.AddedToCart, domain.Purchased:
		// valid
	default:
		return domain.ErrInvalidEventType
	}

	if err := s.engine.RecordEvent(ctx, event); err != nil {
		slog.Error("failed to record user event", "error", err)
		return apperrors.Internal("failed to record event", err)
	}

	return nil
}

// RefreshPopularProducts refreshes the materialized view for popular products.
func (s *RecommendService) RefreshPopularProducts(ctx context.Context) error {
	if err := s.refresher.RefreshPopularProducts(ctx); err != nil {
		slog.Error("failed to refresh popular products", "error", err)
		return apperrors.Internal("failed to refresh popular products", err)
	}
	slog.Info("refreshed popular products materialized view")
	return nil
}
