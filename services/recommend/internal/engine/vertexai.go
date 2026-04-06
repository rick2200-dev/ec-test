package engine

import (
	"context"
	"fmt"

	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
)

// VertexAIEngine implements RecommendEngine using Google Vertex AI Recommendations.
// TODO: Integrate with Vertex AI Predict API for real-time recommendations.
// TODO: Configure Vertex AI model endpoint and feature store.
type VertexAIEngine struct {
	projectID string
}

// NewVertexAIEngine creates a new Vertex AI-based recommendation engine.
func NewVertexAIEngine(projectID string) *VertexAIEngine {
	return &VertexAIEngine{projectID: projectID}
}

// Recommend generates recommendations using Vertex AI.
// TODO: Call Vertex AI Predict API with user/product features.
func (e *VertexAIEngine) Recommend(_ context.Context, _ domain.RecommendRequest) (*domain.RecommendResponse, error) {
	return nil, fmt.Errorf("vertex AI recommendations not implemented yet (project: %s)", e.projectID)
}

// RecordEvent records a user event for Vertex AI model training.
// TODO: Write events to BigQuery or Vertex AI Feature Store for model training.
func (e *VertexAIEngine) RecordEvent(_ context.Context, _ domain.UserEvent) error {
	return fmt.Errorf("vertex AI event recording not implemented yet (project: %s)", e.projectID)
}
