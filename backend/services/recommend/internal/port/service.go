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
	GetRecommendations(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error)
	RecordUserEvent(ctx context.Context, event domain.UserEvent) error
	RefreshPopularProducts(ctx context.Context) error
}
