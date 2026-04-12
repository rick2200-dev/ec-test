// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the recommend service.
package port

import (
	"context"

	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
)

// RecommendUseCase is the driving port (inbound) for recommendation operations.
// Handlers and the subscriber depend on this interface;
// *service.RecommendService satisfies it.
type RecommendUseCase interface {
	// GetRecommendations returns a ranked list of recommended products for the given request context.
	GetRecommendations(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error)
	// RecordUserEvent records a user interaction event (e.g. view, purchase) to improve future recommendations.
	RecordUserEvent(ctx context.Context, event domain.UserEvent) error
	// RefreshPopularProducts rebuilds the popular-products ranking from recent user events;
	// typically called on a schedule.
	RefreshPopularProducts(ctx context.Context) error
}
