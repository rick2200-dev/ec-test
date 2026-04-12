package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	app "github.com/Riku-KANO/ec-test/services/review/internal/app"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
	"github.com/Riku-KANO/ec-test/services/review/internal/port"
)

// ---------------------------------------------------------------------------
// Mock: ReviewStore
// ---------------------------------------------------------------------------

type mockReviewStore struct {
	createFn              func(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error
	getByIDFn             func(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.Review, error)
	updateFn              func(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error
	deleteFn              func(ctx context.Context, tenantID, reviewID uuid.UUID) error
	listByProductFn       func(ctx context.Context, tenantID, productID uuid.UUID, limit, offset int) ([]domain.Review, int, error)
	listBySellerFn        func(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Review, int, error)
	createReplyFn         func(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error
	updateReplyFn         func(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error
	deleteReplyFn         func(ctx context.Context, tenantID, reviewID uuid.UUID) error
	getReplyByReviewFn    func(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.ReviewReply, error)
	getProductRatingFn    func(ctx context.Context, tenantID, productID uuid.UUID) (*domain.ProductRating, error)
	upsertProductRatingFn func(ctx context.Context, tenantID, productID uuid.UUID, ratingDelta, countDelta int) error
}

func (m *mockReviewStore) RunInTx(ctx context.Context, _ uuid.UUID, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (m *mockReviewStore) Create(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, review)
	}
	review.ID = uuid.New()
	return nil
}

func (m *mockReviewStore) GetByID(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.Review, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, tenantID, reviewID)
	}
	return nil, nil
}

func (m *mockReviewStore) Update(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, review)
	}
	return nil
}

func (m *mockReviewStore) Delete(ctx context.Context, tenantID, reviewID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, reviewID)
	}
	return nil
}

func (m *mockReviewStore) ListByProduct(ctx context.Context, tenantID, productID uuid.UUID, limit, offset int) ([]domain.Review, int, error) {
	if m.listByProductFn != nil {
		return m.listByProductFn(ctx, tenantID, productID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockReviewStore) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Review, int, error) {
	if m.listBySellerFn != nil {
		return m.listBySellerFn(ctx, tenantID, sellerID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockReviewStore) CreateReply(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error {
	if m.createReplyFn != nil {
		return m.createReplyFn(ctx, tenantID, reply)
	}
	reply.ID = uuid.New()
	return nil
}

func (m *mockReviewStore) UpdateReply(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error {
	if m.updateReplyFn != nil {
		return m.updateReplyFn(ctx, tenantID, reply)
	}
	return nil
}

func (m *mockReviewStore) DeleteReply(ctx context.Context, tenantID, reviewID uuid.UUID) error {
	if m.deleteReplyFn != nil {
		return m.deleteReplyFn(ctx, tenantID, reviewID)
	}
	return nil
}

func (m *mockReviewStore) GetReplyByReview(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.ReviewReply, error) {
	if m.getReplyByReviewFn != nil {
		return m.getReplyByReviewFn(ctx, tenantID, reviewID)
	}
	return nil, nil
}

func (m *mockReviewStore) GetProductRating(ctx context.Context, tenantID, productID uuid.UUID) (*domain.ProductRating, error) {
	if m.getProductRatingFn != nil {
		return m.getProductRatingFn(ctx, tenantID, productID)
	}
	return nil, nil
}

func (m *mockReviewStore) UpsertProductRating(ctx context.Context, tenantID, productID uuid.UUID, ratingDelta, countDelta int) error {
	if m.upsertProductRatingFn != nil {
		return m.upsertProductRatingFn(ctx, tenantID, productID, ratingDelta, countDelta)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: CatalogClient
// ---------------------------------------------------------------------------

type mockCatalogClient struct {
	getProductFn func(ctx context.Context, tenantID, productID uuid.UUID) (*port.ProductLookup, error)
}

func (m *mockCatalogClient) GetProduct(ctx context.Context, tenantID, productID uuid.UUID) (*port.ProductLookup, error) {
	if m.getProductFn != nil {
		return m.getProductFn(ctx, tenantID, productID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: PurchaseChecker
// ---------------------------------------------------------------------------

type mockPurchaseChecker struct {
	checkPurchaseFn func(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error)
}

func (m *mockPurchaseChecker) CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*port.PurchaseCheckResult, error) {
	if m.checkPurchaseFn != nil {
		return m.checkPurchaseFn(ctx, tenantID, buyerAuth0ID, sellerID, skuID)
	}
	return &port.PurchaseCheckResult{Purchased: false}, nil
}

// ---------------------------------------------------------------------------
// Mock: Publisher (nil / no-op)
// ---------------------------------------------------------------------------

type nilPublisher struct{}

func (n *nilPublisher) Publish(_ context.Context, _ string, _ pubsub.Event) error { return nil }
func (n *nilPublisher) Close() error                                              { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(repo *mockReviewStore, catalog *mockCatalogClient, purchase *mockPurchaseChecker) *app.ReviewService {
	return app.NewReviewService(repo, catalog, purchase, &nilPublisher{})
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

var (
	tenantID  = uuid.New()
	productID = uuid.New()
	sellerID  = uuid.New()
	reviewID  = uuid.New()
	skuID1    = uuid.New()
	skuID2    = uuid.New()
	buyerID   = "auth0|buyer123"
	sellerSub = "auth0|seller456"
)

func defaultProductLookup() *port.ProductLookup {
	return &port.ProductLookup{
		ProductID:   productID,
		SellerID:    sellerID,
		ProductName: "Test Product",
		SKUIDs:      []uuid.UUID{skuID1, skuID2},
	}
}

func defaultReview() *domain.Review {
	return &domain.Review{
		ID:           reviewID,
		TenantID:     tenantID,
		BuyerAuth0ID: buyerID,
		ProductID:    productID,
		SellerID:     sellerID,
		ProductName:  "Test Product",
		Rating:       4,
		Title:        "Great product",
		Body:         "I really liked this product.",
	}
}

func defaultCreateInput() domain.CreateReviewInput {
	return domain.CreateReviewInput{
		ProductID: productID,
		Rating:    4,
		Title:     "Great product",
		Body:      "I really liked this product.",
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateReview
// ---------------------------------------------------------------------------

func TestCreateReview_Success(t *testing.T) {
	repo := &mockReviewStore{}
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	purchase := &mockPurchaseChecker{
		checkPurchaseFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID) (*port.PurchaseCheckResult, error) {
			return &port.PurchaseCheckResult{Purchased: true}, nil
		},
	}
	svc := newTestService(repo, catalog, purchase)

	review, err := svc.CreateReview(context.Background(), tenantID, buyerID, defaultCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review == nil {
		t.Fatal("expected review, got nil")
	}
	if review.Rating != 4 {
		t.Errorf("expected rating 4, got %d", review.Rating)
	}
	if review.Title != "Great product" {
		t.Errorf("expected title 'Great product', got %q", review.Title)
	}
	if review.SellerID != sellerID {
		t.Errorf("expected seller_id %s, got %s", sellerID, review.SellerID)
	}
	if review.ProductName != "Test Product" {
		t.Errorf("expected product_name 'Test Product', got %q", review.ProductName)
	}
}

func TestCreateReview_InvalidRating_Zero(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.Rating = 0

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if !errors.Is(err, domain.ErrInvalidRating) {
		t.Fatalf("expected ErrInvalidRating, got %v", err)
	}
}

func TestCreateReview_InvalidRating_Six(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.Rating = 6

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if !errors.Is(err, domain.ErrInvalidRating) {
		t.Fatalf("expected ErrInvalidRating, got %v", err)
	}
}

func TestCreateReview_EmptyBuyer(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, err := svc.CreateReview(context.Background(), tenantID, "", defaultCreateInput())
	if err == nil {
		t.Fatal("expected error for empty buyer")
	}
	if !strings.Contains(err.Error(), "buyer_auth0_id is required") {
		t.Fatalf("expected buyer_auth0_id error, got %v", err)
	}
}

func TestCreateReview_EmptyProductID(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.ProductID = uuid.Nil

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for empty product_id")
	}
	if !strings.Contains(err.Error(), "product_id is required") {
		t.Fatalf("expected product_id error, got %v", err)
	}
}

func TestCreateReview_EmptyTitle(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.Title = ""

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("expected title error, got %v", err)
	}
}

func TestCreateReview_EmptyBody(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.Body = "   " // whitespace-only should be treated as empty

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "body is required") {
		t.Fatalf("expected body error, got %v", err)
	}
}

func TestCreateReview_PurchaseNotFound(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	purchase := &mockPurchaseChecker{
		checkPurchaseFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID) (*port.PurchaseCheckResult, error) {
			return &port.PurchaseCheckResult{Purchased: false}, nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, purchase)

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, defaultCreateInput())
	if !errors.Is(err, domain.ErrPurchaseRequired) {
		t.Fatalf("expected ErrPurchaseRequired, got %v", err)
	}
}

func TestCreateReview_CatalogError(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return nil, errors.New("catalog unavailable")
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, &mockPurchaseChecker{})

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, defaultCreateInput())
	if err == nil {
		t.Fatal("expected error when catalog fails")
	}
	if !strings.Contains(err.Error(), "catalog unavailable") {
		t.Fatalf("expected catalog error message, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: GetReview
// ---------------------------------------------------------------------------

func TestGetReview_Success(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	review, err := svc.GetReview(context.Background(), tenantID, reviewID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review == nil {
		t.Fatal("expected review, got nil")
	}
	if review.ID != reviewID {
		t.Errorf("expected review_id %s, got %s", reviewID, review.ID)
	}
}

func TestGetReview_NotFound(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, err := svc.GetReview(context.Background(), tenantID, reviewID)
	if !errors.Is(err, domain.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateReview
// ---------------------------------------------------------------------------

func TestUpdateReview_SuccessWithRatingChange(t *testing.T) {
	existingReview := defaultReview()
	existingReview.Rating = 3

	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			// Return a copy so mutations don't affect the original.
			r := *existingReview
			return &r, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{
		Rating: intPtr(5),
		Title:  strPtr("Updated title"),
	}
	review, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Rating != 5 {
		t.Errorf("expected rating 5, got %d", review.Rating)
	}
	if review.Title != "Updated title" {
		t.Errorf("expected title 'Updated title', got %q", review.Title)
	}
}

func TestUpdateReview_NotFound(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Rating: intPtr(3)}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if !errors.Is(err, domain.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestUpdateReview_NotOwner(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Rating: intPtr(3)}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, "auth0|someone_else", in)
	if !errors.Is(err, domain.ErrNotReviewOwner) {
		t.Fatalf("expected ErrNotReviewOwner, got %v", err)
	}
}

func TestUpdateReview_InvalidRating(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Rating: intPtr(0)}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if !errors.Is(err, domain.ErrInvalidRating) {
		t.Fatalf("expected ErrInvalidRating, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: DeleteReview
// ---------------------------------------------------------------------------

func TestDeleteReview_Success(t *testing.T) {
	deleteCalled := false
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	err := svc.DeleteReview(context.Background(), tenantID, reviewID, buyerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected Delete to be called")
	}
}

func TestDeleteReview_NotFound(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	err := svc.DeleteReview(context.Background(), tenantID, reviewID, buyerID)
	if !errors.Is(err, domain.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestDeleteReview_NotOwner(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	err := svc.DeleteReview(context.Background(), tenantID, reviewID, "auth0|someone_else")
	if !errors.Is(err, domain.ErrNotReviewOwner) {
		t.Fatalf("expected ErrNotReviewOwner, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: ListByProduct
// ---------------------------------------------------------------------------

func TestListByProduct_Success(t *testing.T) {
	expected := []domain.Review{*defaultReview()}
	repo := &mockReviewStore{
		listByProductFn: func(_ context.Context, _, _ uuid.UUID, limit, offset int) ([]domain.Review, int, error) {
			if limit != 10 {
				t.Errorf("expected limit 10, got %d", limit)
			}
			if offset != 5 {
				t.Errorf("expected offset 5, got %d", offset)
			}
			return expected, 1, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	items, total, err := svc.ListByProduct(context.Background(), tenantID, productID, 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
}

func TestListByProduct_DefaultLimitOffset(t *testing.T) {
	repo := &mockReviewStore{
		listByProductFn: func(_ context.Context, _, _ uuid.UUID, limit, offset int) ([]domain.Review, int, error) {
			if limit != 20 {
				t.Errorf("expected default limit 20, got %d", limit)
			}
			if offset != 0 {
				t.Errorf("expected default offset 0, got %d", offset)
			}
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListByProduct(context.Background(), tenantID, productID, 0, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: GetProductRating
// ---------------------------------------------------------------------------

func TestGetProductRating_Success(t *testing.T) {
	repo := &mockReviewStore{
		getProductRatingFn: func(_ context.Context, _, _ uuid.UUID) (*domain.ProductRating, error) {
			return &domain.ProductRating{
				TenantID:      tenantID,
				ProductID:     productID,
				AverageRating: 4.5,
				ReviewCount:   10,
			}, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	rating, err := svc.GetProductRating(context.Background(), tenantID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rating.AverageRating != 4.5 {
		t.Errorf("expected avg 4.5, got %f", rating.AverageRating)
	}
	if rating.ReviewCount != 10 {
		t.Errorf("expected count 10, got %d", rating.ReviewCount)
	}
}

func TestGetProductRating_NilReturnsZero(t *testing.T) {
	repo := &mockReviewStore{
		getProductRatingFn: func(_ context.Context, _, _ uuid.UUID) (*domain.ProductRating, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	rating, err := svc.GetProductRating(context.Background(), tenantID, productID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rating == nil {
		t.Fatal("expected non-nil rating")
	}
	if rating.AverageRating != 0 {
		t.Errorf("expected avg 0, got %f", rating.AverageRating)
	}
	if rating.ReviewCount != 0 {
		t.Errorf("expected count 0, got %d", rating.ReviewCount)
	}
	if rating.TenantID != tenantID {
		t.Errorf("expected tenant_id %s, got %s", tenantID, rating.TenantID)
	}
	if rating.ProductID != productID {
		t.Errorf("expected product_id %s, got %s", productID, rating.ProductID)
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateReply
// ---------------------------------------------------------------------------

func TestCreateReply_Success(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.CreateReplyInput{Body: "Thank you for the review!"}
	reply, err := svc.CreateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply == nil {
		t.Fatal("expected reply, got nil")
	}
	if reply.ReviewID != reviewID {
		t.Errorf("expected review_id %s, got %s", reviewID, reply.ReviewID)
	}
	if reply.Body != "Thank you for the review!" {
		t.Errorf("unexpected body %q", reply.Body)
	}
}

func TestCreateReply_EmptyBody(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.CreateReplyInput{Body: "  "}
	_, err := svc.CreateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "body is required") {
		t.Fatalf("expected body error, got %v", err)
	}
}

func TestCreateReply_NotSellerOfProduct(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	otherSellerID := uuid.New()
	in := domain.CreateReplyInput{Body: "Reply body"}
	_, err := svc.CreateReply(context.Background(), tenantID, reviewID, sellerSub, otherSellerID, in)
	if !errors.Is(err, domain.ErrNotSellerOfProduct) {
		t.Fatalf("expected ErrNotSellerOfProduct, got %v", err)
	}
}

func TestCreateReply_ReviewNotFound(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.CreateReplyInput{Body: "Reply body"}
	_, err := svc.CreateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if !errors.Is(err, domain.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: DeleteReply
// ---------------------------------------------------------------------------

func TestDeleteReply_Success(t *testing.T) {
	deleteReplyCalled := false
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
		deleteReplyFn: func(_ context.Context, _, _ uuid.UUID) error {
			deleteReplyCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	err := svc.DeleteReply(context.Background(), tenantID, reviewID, sellerID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleteReplyCalled {
		t.Error("expected DeleteReply to be called")
	}
}

func TestDeleteReply_ReviewNotFound(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	err := svc.DeleteReply(context.Background(), tenantID, reviewID, sellerID)
	if !errors.Is(err, domain.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestDeleteReply_NotSeller(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	otherSellerID := uuid.New()
	err := svc.DeleteReply(context.Background(), tenantID, reviewID, otherSellerID)
	if !errors.Is(err, domain.ErrNotSellerOfProduct) {
		t.Fatalf("expected ErrNotSellerOfProduct, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateReply
// ---------------------------------------------------------------------------

func TestUpdateReply_Success(t *testing.T) {
	updateReplyCalled := false
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
		updateReplyFn: func(_ context.Context, _ uuid.UUID, reply *domain.ReviewReply) error {
			updateReplyCalled = true
			if reply.ReviewID != reviewID {
				t.Errorf("expected review_id %s, got %s", reviewID, reply.ReviewID)
			}
			if reply.Body != "Updated reply body" {
				t.Errorf("expected body 'Updated reply body', got %q", reply.Body)
			}
			return nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReplyInput{Body: "Updated reply body"}
	reply, err := svc.UpdateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply == nil {
		t.Fatal("expected reply, got nil")
	}
	if !updateReplyCalled {
		t.Error("expected UpdateReply to be called on repo")
	}
}

func TestUpdateReply_EmptyBody(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReplyInput{Body: "   "}
	_, err := svc.UpdateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "body is required") {
		t.Fatalf("expected body error, got %v", err)
	}
}

func TestUpdateReply_ReviewNotFound(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReplyInput{Body: "Some reply"}
	_, err := svc.UpdateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if !errors.Is(err, domain.ErrReviewNotFound) {
		t.Fatalf("expected ErrReviewNotFound, got %v", err)
	}
}

func TestUpdateReply_NotSellerOfProduct(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	otherSellerID := uuid.New()
	in := domain.UpdateReplyInput{Body: "Some reply"}
	_, err := svc.UpdateReply(context.Background(), tenantID, reviewID, sellerSub, otherSellerID, in)
	if !errors.Is(err, domain.ErrNotSellerOfProduct) {
		t.Fatalf("expected ErrNotSellerOfProduct, got %v", err)
	}
}

func TestUpdateReply_BodyTooLong(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	longBody := strings.Repeat("x", 2001)
	in := domain.UpdateReplyInput{Body: longBody}
	_, err := svc.UpdateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err == nil {
		t.Fatal("expected error for body too long")
	}
	if !strings.Contains(err.Error(), "body must be at most 2000 characters") {
		t.Fatalf("expected body length error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: ListForSeller
// ---------------------------------------------------------------------------

func TestListForSeller_Success(t *testing.T) {
	expected := []domain.Review{*defaultReview()}
	repo := &mockReviewStore{
		listBySellerFn: func(_ context.Context, _, sid uuid.UUID, limit, offset int) ([]domain.Review, int, error) {
			if sid != sellerID {
				t.Errorf("expected seller_id %s, got %s", sellerID, sid)
			}
			if limit != 10 {
				t.Errorf("expected limit 10, got %d", limit)
			}
			if offset != 5 {
				t.Errorf("expected offset 5, got %d", offset)
			}
			return expected, 1, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	items, total, err := svc.ListForSeller(context.Background(), tenantID, sellerID, 10, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
}

func TestListForSeller_NilSellerID(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListForSeller(context.Background(), tenantID, uuid.Nil, 10, 0)
	if err == nil {
		t.Fatal("expected error for nil seller_id")
	}
	if !strings.Contains(err.Error(), "seller_id is required") {
		t.Fatalf("expected seller_id error, got %v", err)
	}
}

func TestListForSeller_DefaultLimitOffset(t *testing.T) {
	repo := &mockReviewStore{
		listBySellerFn: func(_ context.Context, _, _ uuid.UUID, limit, offset int) ([]domain.Review, int, error) {
			if limit != 20 {
				t.Errorf("expected default limit 20, got %d", limit)
			}
			if offset != 0 {
				t.Errorf("expected default offset 0, got %d", offset)
			}
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListForSeller(context.Background(), tenantID, sellerID, 0, -1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListForSeller_LimitExceeds100(t *testing.T) {
	repo := &mockReviewStore{
		listBySellerFn: func(_ context.Context, _, _ uuid.UUID, limit, _ int) ([]domain.Review, int, error) {
			if limit != 20 {
				t.Errorf("expected default limit 20 when >100, got %d", limit)
			}
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListForSeller(context.Background(), tenantID, sellerID, 101, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateReview - additional edge cases
// ---------------------------------------------------------------------------

func TestCreateReview_TitleTooLong(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.Title = strings.Repeat("a", 256)

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for title too long")
	}
	if !strings.Contains(err.Error(), "title must be at most 255 characters") {
		t.Fatalf("expected title length error, got %v", err)
	}
}

func TestCreateReview_BodyTooLong(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, &mockPurchaseChecker{})

	in := defaultCreateInput()
	in.Body = strings.Repeat("b", 4001)

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for body too long")
	}
	if !strings.Contains(err.Error(), "body must be at most 4000 characters") {
		t.Fatalf("expected body length error, got %v", err)
	}
}

func TestCreateReview_PurchaseCheckPartialError(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	callCount := 0
	purchase := &mockPurchaseChecker{
		checkPurchaseFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID) (*port.PurchaseCheckResult, error) {
			callCount++
			// First SKU errors, second SKU returns not-purchased.
			if callCount == 1 {
				return nil, errors.New("order service timeout")
			}
			return &port.PurchaseCheckResult{Purchased: false}, nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, purchase)

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, defaultCreateInput())
	if err == nil {
		t.Fatal("expected error for partial purchase check failure")
	}
	if !strings.Contains(err.Error(), "purchase check partially failed") {
		t.Fatalf("expected partial failure error, got %v", err)
	}
}

func TestCreateReview_PurchaseCheckAllErrors(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	purchase := &mockPurchaseChecker{
		checkPurchaseFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID) (*port.PurchaseCheckResult, error) {
			return nil, errors.New("order service down")
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, purchase)

	_, err := svc.CreateReview(context.Background(), tenantID, buyerID, defaultCreateInput())
	if err == nil {
		t.Fatal("expected error when all purchase checks fail")
	}
	if !strings.Contains(err.Error(), "purchase check partially failed") {
		t.Fatalf("expected partial failure error, got %v", err)
	}
}

func TestCreateReview_PurchaseCheckFirstSKUSucceeds(t *testing.T) {
	catalog := &mockCatalogClient{
		getProductFn: func(_ context.Context, _, _ uuid.UUID) (*port.ProductLookup, error) {
			return defaultProductLookup(), nil
		},
	}
	callCount := 0
	purchase := &mockPurchaseChecker{
		checkPurchaseFn: func(_ context.Context, _ uuid.UUID, _ string, _, _ uuid.UUID) (*port.PurchaseCheckResult, error) {
			callCount++
			// First SKU purchased, second should not be called.
			return &port.PurchaseCheckResult{Purchased: true}, nil
		},
	}
	svc := newTestService(&mockReviewStore{}, catalog, purchase)

	review, err := svc.CreateReview(context.Background(), tenantID, buyerID, defaultCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review == nil {
		t.Fatal("expected review, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected purchase check to stop after first success, but was called %d times", callCount)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateReview - additional edge cases
// ---------------------------------------------------------------------------

func TestUpdateReview_EmptyTitleUpdate(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Title: strPtr("   ")}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("expected title error, got %v", err)
	}
}

func TestUpdateReview_EmptyBodyUpdate(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Body: strPtr("  ")}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
	if !strings.Contains(err.Error(), "body is required") {
		t.Fatalf("expected body error, got %v", err)
	}
}

func TestUpdateReview_TitleTooLong(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	longTitle := strings.Repeat("a", 256)
	in := domain.UpdateReviewInput{Title: strPtr(longTitle)}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for title too long")
	}
	if !strings.Contains(err.Error(), "title must be at most 255 characters") {
		t.Fatalf("expected title length error, got %v", err)
	}
}

func TestUpdateReview_BodyTooLong(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	longBody := strings.Repeat("b", 4001)
	in := domain.UpdateReviewInput{Body: strPtr(longBody)}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err == nil {
		t.Fatal("expected error for body too long")
	}
	if !strings.Contains(err.Error(), "body must be at most 4000 characters") {
		t.Fatalf("expected body length error, got %v", err)
	}
}

func TestUpdateReview_InvalidRatingHigh(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Rating: intPtr(6)}
	_, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if !errors.Is(err, domain.ErrInvalidRating) {
		t.Fatalf("expected ErrInvalidRating, got %v", err)
	}
}

func TestUpdateReview_BodyOnly(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReviewInput{Body: strPtr("Updated body text")}
	review, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Body != "Updated body text" {
		t.Errorf("expected body 'Updated body text', got %q", review.Body)
	}
	// Rating should remain unchanged.
	if review.Rating != 4 {
		t.Errorf("expected rating unchanged at 4, got %d", review.Rating)
	}
}

func TestUpdateReview_NoFieldsChanged(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	// Empty input — no fields set.
	in := domain.UpdateReviewInput{}
	review, err := svc.UpdateReview(context.Background(), tenantID, reviewID, buyerID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if review.Rating != 4 {
		t.Errorf("expected unchanged rating 4, got %d", review.Rating)
	}
	if review.Title != "Great product" {
		t.Errorf("expected unchanged title, got %q", review.Title)
	}
}

// ---------------------------------------------------------------------------
// Tests: CreateReply - additional edge cases
// ---------------------------------------------------------------------------

func TestCreateReply_BodyTooLong(t *testing.T) {
	svc := newTestService(&mockReviewStore{}, &mockCatalogClient{}, &mockPurchaseChecker{})

	longBody := strings.Repeat("x", 2001)
	in := domain.CreateReplyInput{Body: longBody}
	_, err := svc.CreateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err == nil {
		t.Fatal("expected error for body too long")
	}
	if !strings.Contains(err.Error(), "body must be at most 2000 characters") {
		t.Fatalf("expected body length error, got %v", err)
	}
}

func TestCreateReply_BodyExactly2000(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return defaultReview(), nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	body := strings.Repeat("y", 2000)
	in := domain.CreateReplyInput{Body: body}
	reply, err := svc.CreateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply == nil {
		t.Fatal("expected reply, got nil")
	}
	if len(reply.Body) != 2000 {
		t.Errorf("expected body length 2000, got %d", len(reply.Body))
	}
}

// ---------------------------------------------------------------------------
// Tests: GetReview - repo error
// ---------------------------------------------------------------------------

func TestGetReview_RepoError(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, errors.New("db connection lost")
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, err := svc.GetReview(context.Background(), tenantID, reviewID)
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
	if !strings.Contains(err.Error(), "failed to load review") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: ListByProduct - repo error
// ---------------------------------------------------------------------------

func TestListByProduct_RepoError(t *testing.T) {
	repo := &mockReviewStore{
		listByProductFn: func(_ context.Context, _, _ uuid.UUID, _, _ int) ([]domain.Review, int, error) {
			return nil, 0, errors.New("db timeout")
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListByProduct(context.Background(), tenantID, productID, 10, 0)
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
	if !strings.Contains(err.Error(), "failed to list reviews") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestListByProduct_LimitExceeds100(t *testing.T) {
	repo := &mockReviewStore{
		listByProductFn: func(_ context.Context, _, _ uuid.UUID, limit, _ int) ([]domain.Review, int, error) {
			if limit != 20 {
				t.Errorf("expected default limit 20 when >100, got %d", limit)
			}
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListByProduct(context.Background(), tenantID, productID, 101, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: ListForSeller - repo error
// ---------------------------------------------------------------------------

func TestListForSeller_RepoError(t *testing.T) {
	repo := &mockReviewStore{
		listBySellerFn: func(_ context.Context, _, _ uuid.UUID, _, _ int) ([]domain.Review, int, error) {
			return nil, 0, errors.New("db timeout")
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, _, err := svc.ListForSeller(context.Background(), tenantID, sellerID, 10, 0)
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
	if !strings.Contains(err.Error(), "failed to list reviews") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: GetProductRating - repo error
// ---------------------------------------------------------------------------

func TestGetProductRating_RepoError(t *testing.T) {
	repo := &mockReviewStore{
		getProductRatingFn: func(_ context.Context, _, _ uuid.UUID) (*domain.ProductRating, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	_, err := svc.GetProductRating(context.Background(), tenantID, productID)
	if err == nil {
		t.Fatal("expected error when repo fails")
	}
	if !strings.Contains(err.Error(), "failed to load product rating") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: DeleteReview - repo error on GetByID
// ---------------------------------------------------------------------------

func TestDeleteReview_RepoGetError(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	err := svc.DeleteReview(context.Background(), tenantID, reviewID, buyerID)
	if err == nil {
		t.Fatal("expected error when repo GetByID fails")
	}
	if !strings.Contains(err.Error(), "failed to load review") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Tests: UpdateReply - repo error on GetByID
// ---------------------------------------------------------------------------

func TestUpdateReply_RepoGetError(t *testing.T) {
	repo := &mockReviewStore{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Review, error) {
			return nil, errors.New("db error")
		},
	}
	svc := newTestService(repo, &mockCatalogClient{}, &mockPurchaseChecker{})

	in := domain.UpdateReplyInput{Body: "Some reply"}
	_, err := svc.UpdateReply(context.Background(), tenantID, reviewID, sellerSub, sellerID, in)
	if err == nil {
		t.Fatal("expected error when repo GetByID fails")
	}
	if !strings.Contains(err.Error(), "failed to load review") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
