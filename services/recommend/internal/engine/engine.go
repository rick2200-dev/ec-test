package engine

import (
	"context"

	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
)

// RecommendEngine is the interface for generating product recommendations.
type RecommendEngine interface {
	// Recommend generates product recommendations based on the request parameters.
	Recommend(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error)

	// RecordEvent records a user behavior event for future recommendation generation.
	RecordEvent(ctx context.Context, event domain.UserEvent) error
}
