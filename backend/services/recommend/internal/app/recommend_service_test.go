package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/engine"
)

// ---- test doubles ----

type mockEngine struct {
	recommendFn   func(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error)
	recordEventFn func(ctx context.Context, event domain.UserEvent) error
}

func (m *mockEngine) Recommend(ctx context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
	if m.recommendFn != nil {
		return m.recommendFn(ctx, req)
	}
	return &domain.RecommendResponse{Products: []domain.RecommendedProduct{}}, nil
}

func (m *mockEngine) RecordEvent(ctx context.Context, event domain.UserEvent) error {
	if m.recordEventFn != nil {
		return m.recordEventFn(ctx, event)
	}
	return nil
}

type mockRefresher struct{ err error }

func (m *mockRefresher) RefreshPopularProducts(_ context.Context) error { return m.err }

// compile-time interface checks
var _ engine.RecommendEngine = (*mockEngine)(nil)
var _ ViewRefresher = (*mockRefresher)(nil)

func newSvc(eng *mockEngine, ref *mockRefresher) *RecommendService {
	return NewRecommendService(eng, ref)
}

// ---- GetRecommendations ----

func TestGetRecommendations_Success(t *testing.T) {
	tid := uuid.New()
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	resp, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID: tid,
		Type:     domain.Popular,
		Limit:    5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestGetRecommendations_MissingTenantID(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	_, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		Type: domain.Popular,
	})
	if !errors.Is(err, domain.ErrMissingTenantID) {
		t.Errorf("want ErrMissingTenantID, got %v", err)
	}
}

func TestGetRecommendations_LimitClampedToDefault(t *testing.T) {
	tid := uuid.New()
	var capturedLimit int
	eng := &mockEngine{
		recommendFn: func(_ context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
			capturedLimit = req.Limit
			return &domain.RecommendResponse{}, nil
		},
	}
	svc := newSvc(eng, &mockRefresher{})

	_, _ = svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID: tid,
		Type:     domain.Popular,
		Limit:    0,
	})
	if capturedLimit != 10 {
		t.Errorf("limit = %d, want 10 (default)", capturedLimit)
	}

	_, _ = svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID: tid,
		Type:     domain.Popular,
		Limit:    -5,
	})
	if capturedLimit != 10 {
		t.Errorf("limit = %d, want 10 (negative clamped)", capturedLimit)
	}
}

func TestGetRecommendations_LimitClampedToMax(t *testing.T) {
	tid := uuid.New()
	var capturedLimit int
	eng := &mockEngine{
		recommendFn: func(_ context.Context, req domain.RecommendRequest) (*domain.RecommendResponse, error) {
			capturedLimit = req.Limit
			return &domain.RecommendResponse{}, nil
		},
	}
	svc := newSvc(eng, &mockRefresher{})

	_, _ = svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID: tid,
		Type:     domain.Popular,
		Limit:    999,
	})
	if capturedLimit != 100 {
		t.Errorf("limit = %d, want 100 (max)", capturedLimit)
	}
}

func TestGetRecommendations_InvalidType(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	_, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID: uuid.New(),
		Type:     "unknown_type",
		Limit:    10,
	})
	if !errors.Is(err, domain.ErrInvalidRecommendationType) {
		t.Errorf("want ErrInvalidRecommendationType, got %v", err)
	}
}

func TestGetRecommendations_SimilarMissingProductID(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	_, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID:  uuid.New(),
		Type:      domain.Similar,
		Limit:     10,
		ProductID: nil,
	})
	if !errors.Is(err, domain.ErrMissingProductID) {
		t.Errorf("want ErrMissingProductID, got %v", err)
	}
}

func TestGetRecommendations_FrequentlyBoughtTogetherMissingProductID(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	_, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID:  uuid.New(),
		Type:      domain.FrequentlyBoughtTogether,
		Limit:     10,
		ProductID: nil,
	})
	if !errors.Is(err, domain.ErrMissingProductID) {
		t.Errorf("want ErrMissingProductID, got %v", err)
	}
}

func TestGetRecommendations_SimilarWithProductID(t *testing.T) {
	pid := uuid.New()
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	_, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID:  uuid.New(),
		Type:      domain.Similar,
		Limit:     5,
		ProductID: &pid,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetRecommendations_EngineError(t *testing.T) {
	engineErr := errors.New("engine unavailable")
	eng := &mockEngine{
		recommendFn: func(_ context.Context, _ domain.RecommendRequest) (*domain.RecommendResponse, error) {
			return nil, engineErr
		},
	}
	svc := newSvc(eng, &mockRefresher{})

	_, err := svc.GetRecommendations(context.Background(), domain.RecommendRequest{
		TenantID: uuid.New(),
		Type:     domain.Popular,
		Limit:    5,
	})

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *apperrors.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Errorf("status = %d, want 500", appErr.Status)
	}
}

// ---- RecordUserEvent ----

func TestRecordUserEvent_Success(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
		TenantID:  uuid.New(),
		UserID:    "auth0|u1",
		ProductID: uuid.New(),
		EventType: domain.ProductViewed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecordUserEvent_MissingTenantID(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
		UserID:    "auth0|u1",
		ProductID: uuid.New(),
		EventType: domain.ProductViewed,
	})
	if !errors.Is(err, domain.ErrMissingTenantID) {
		t.Errorf("want ErrMissingTenantID, got %v", err)
	}
}

func TestRecordUserEvent_MissingUserID(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
		TenantID:  uuid.New(),
		ProductID: uuid.New(),
		EventType: domain.ProductViewed,
	})
	if !errors.Is(err, domain.ErrMissingUserID) {
		t.Errorf("want ErrMissingUserID, got %v", err)
	}
}

func TestRecordUserEvent_MissingProductID(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
		TenantID:  uuid.New(),
		UserID:    "auth0|u1",
		EventType: domain.ProductViewed,
	})
	if !errors.Is(err, domain.ErrMissingProductID) {
		t.Errorf("want ErrMissingProductID, got %v", err)
	}
}

func TestRecordUserEvent_InvalidEventType(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
		TenantID:  uuid.New(),
		UserID:    "auth0|u1",
		ProductID: uuid.New(),
		EventType: "unknown_event",
	})
	if !errors.Is(err, domain.ErrInvalidEventType) {
		t.Errorf("want ErrInvalidEventType, got %v", err)
	}
}

func TestRecordUserEvent_AllValidEventTypes(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})

	for _, et := range []domain.UserEventType{domain.ProductViewed, domain.AddedToCart, domain.Purchased} {
		err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
			TenantID:  uuid.New(),
			UserID:    "auth0|u1",
			ProductID: uuid.New(),
			EventType: et,
		})
		if err != nil {
			t.Errorf("event type %q: unexpected error: %v", et, err)
		}
	}
}

func TestRecordUserEvent_EngineError(t *testing.T) {
	eng := &mockEngine{
		recordEventFn: func(_ context.Context, _ domain.UserEvent) error {
			return errors.New("record failed")
		},
	}
	svc := newSvc(eng, &mockRefresher{})

	err := svc.RecordUserEvent(context.Background(), domain.UserEvent{
		TenantID:  uuid.New(),
		UserID:    "auth0|u1",
		ProductID: uuid.New(),
		EventType: domain.ProductViewed,
	})

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *apperrors.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Errorf("status = %d, want 500", appErr.Status)
	}
}

// ---- RefreshPopularProducts ----

func TestRefreshPopularProducts_Success(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{})
	if err := svc.RefreshPopularProducts(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshPopularProducts_Error(t *testing.T) {
	svc := newSvc(&mockEngine{}, &mockRefresher{err: errors.New("db down")})

	err := svc.RefreshPopularProducts(context.Background())

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *apperrors.AppError, got %T: %v", err, err)
	}
	if appErr.Status != 500 {
		t.Errorf("status = %d, want 500", appErr.Status)
	}
}
