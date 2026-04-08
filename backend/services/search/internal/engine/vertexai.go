package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
)

// VertexAIEngine implements SearchEngine using Google Vertex AI Search for Commerce.
// TODO: Integrate with Vertex AI Retail Search API.
type VertexAIEngine struct {
	projectID string
}

// NewVertexAIEngine creates a new VertexAIEngine.
func NewVertexAIEngine(projectID string) *VertexAIEngine {
	return &VertexAIEngine{projectID: projectID}
}

// Search queries Vertex AI Search for Commerce.
// TODO: Implement Vertex AI Retail Search API call with:
//   - ServingConfig for the tenant
//   - UserEvent for personalization
//   - Facet specifications
//   - Boost/bury rules
func (e *VertexAIEngine) Search(_ context.Context, _ domain.SearchRequest) (*domain.SearchResult, error) {
	return nil, fmt.Errorf("vertex AI search not implemented: project=%s", e.projectID)
}

// IndexProduct sends a product update to the Vertex AI product catalog.
// TODO: Implement using Vertex AI Retail API ProductService.ImportProducts or
// ProductService.UpdateProduct for real-time updates.
func (e *VertexAIEngine) IndexProduct(_ context.Context, _ domain.ProductEvent) error {
	return fmt.Errorf("vertex AI index not implemented: project=%s", e.projectID)
}

// DeleteProduct removes a product from the Vertex AI product catalog.
// TODO: Implement using Vertex AI Retail API ProductService.DeleteProduct.
func (e *VertexAIEngine) DeleteProduct(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return fmt.Errorf("vertex AI delete not implemented: project=%s", e.projectID)
}
