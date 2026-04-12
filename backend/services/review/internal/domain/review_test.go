package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	domain "github.com/Riku-KANO/ec-test/services/review/internal/domain"
)

func TestDomainErrors_NotNil(t *testing.T) {
	t.Parallel()

	errs := []struct {
		name string
		err  error
	}{
		{"ErrReviewNotFound", domain.ErrReviewNotFound},
		{"ErrReplyNotFound", domain.ErrReplyNotFound},
		{"ErrAlreadyReviewed", domain.ErrAlreadyReviewed},
		{"ErrAlreadyReplied", domain.ErrAlreadyReplied},
		{"ErrPurchaseRequired", domain.ErrPurchaseRequired},
		{"ErrNotReviewOwner", domain.ErrNotReviewOwner},
		{"ErrNotSellerOfProduct", domain.ErrNotSellerOfProduct},
		{"ErrInvalidRating", domain.ErrInvalidRating},
		{"ErrProductNotFound", domain.ErrProductNotFound},
	}

	for _, tc := range errs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatalf("%s should not be nil", tc.name)
			}
		})
	}
}

func TestDomainErrors_Distinct(t *testing.T) {
	t.Parallel()

	errs := []error{
		domain.ErrReviewNotFound,
		domain.ErrReplyNotFound,
		domain.ErrAlreadyReviewed,
		domain.ErrAlreadyReplied,
		domain.ErrPurchaseRequired,
		domain.ErrNotReviewOwner,
		domain.ErrNotSellerOfProduct,
		domain.ErrInvalidRating,
		domain.ErrProductNotFound,
	}

	seen := make(map[string]int)
	for i, err := range errs {
		msg := err.Error()
		if prev, ok := seen[msg]; ok {
			t.Fatalf("error at index %d has the same message as error at index %d: %q", i, prev, msg)
		}
		seen[msg] = i
	}
}

func TestReview_WithReply(t *testing.T) {
	t.Parallel()

	now := time.Now()
	reviewID := uuid.New()
	tenantID := uuid.New()

	t.Run("without reply", func(t *testing.T) {
		productID := uuid.New()
		sellerID := uuid.New()
		r := domain.Review{
			ID:           reviewID,
			TenantID:     tenantID,
			BuyerAuth0ID: "auth0|buyer1",
			ProductID:    productID,
			SellerID:     sellerID,
			ProductName:  "Test Product",
			Rating:       4,
			Title:        "Great product",
			Body:         "I really liked it.",
			CreatedAt:    now,
			UpdatedAt:    now,
			Reply:        nil,
		}

		if r.Reply != nil {
			t.Fatal("expected Reply to be nil")
		}
		if r.ID != reviewID {
			t.Fatalf("expected ID %s, got %s", reviewID, r.ID)
		}
		if r.TenantID != tenantID {
			t.Fatalf("expected TenantID %s, got %s", tenantID, r.TenantID)
		}
		if r.BuyerAuth0ID != "auth0|buyer1" {
			t.Fatalf("expected BuyerAuth0ID %q, got %q", "auth0|buyer1", r.BuyerAuth0ID)
		}
		if r.ProductID != productID {
			t.Fatalf("expected ProductID %s, got %s", productID, r.ProductID)
		}
		if r.SellerID != sellerID {
			t.Fatalf("expected SellerID %s, got %s", sellerID, r.SellerID)
		}
		if r.ProductName != "Test Product" {
			t.Fatalf("expected ProductName %q, got %q", "Test Product", r.ProductName)
		}
		if r.Rating != 4 {
			t.Fatalf("expected Rating 4, got %d", r.Rating)
		}
		if r.Title != "Great product" {
			t.Fatalf("expected Title %q, got %q", "Great product", r.Title)
		}
		if r.Body != "I really liked it." {
			t.Fatalf("expected Body %q, got %q", "I really liked it.", r.Body)
		}
		if r.CreatedAt != now {
			t.Fatalf("expected CreatedAt %v, got %v", now, r.CreatedAt)
		}
		if r.UpdatedAt != now {
			t.Fatalf("expected UpdatedAt %v, got %v", now, r.UpdatedAt)
		}
	})

	t.Run("with reply", func(t *testing.T) {
		replyID := uuid.New()
		reply := &domain.ReviewReply{
			ID:            replyID,
			TenantID:      tenantID,
			ReviewID:      reviewID,
			SellerAuth0ID: "auth0|seller1",
			Body:          "Thank you for your feedback!",
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		r := domain.Review{
			ID:    reviewID,
			Rating: 5,
			Reply: reply,
		}

		if r.Reply == nil {
			t.Fatal("expected Reply to be non-nil")
		}
		if r.Reply.ID != replyID {
			t.Fatalf("expected Reply.ID %s, got %s", replyID, r.Reply.ID)
		}
		if r.Reply.ReviewID != reviewID {
			t.Fatalf("expected Reply.ReviewID %s, got %s", reviewID, r.Reply.ReviewID)
		}
		if r.Reply.TenantID != tenantID {
			t.Fatalf("expected Reply.TenantID %s, got %s", tenantID, r.Reply.TenantID)
		}
		if r.Reply.SellerAuth0ID != "auth0|seller1" {
			t.Fatalf("expected SellerAuth0ID %q, got %q", "auth0|seller1", r.Reply.SellerAuth0ID)
		}
		if r.Reply.Body != "Thank you for your feedback!" {
			t.Fatalf("unexpected Reply.Body: %q", r.Reply.Body)
		}
		if r.Reply.CreatedAt != now {
			t.Fatalf("expected Reply.CreatedAt %v, got %v", now, r.Reply.CreatedAt)
		}
		if r.Reply.UpdatedAt != now {
			t.Fatalf("expected Reply.UpdatedAt %v, got %v", now, r.Reply.UpdatedAt)
		}
		if r.Rating != 5 {
			t.Fatalf("expected Rating 5, got %d", r.Rating)
		}
	})
}

func TestProductRating_ZeroValue(t *testing.T) {
	t.Parallel()

	var pr domain.ProductRating

	if pr.TenantID != uuid.Nil {
		t.Fatalf("expected zero TenantID, got %s", pr.TenantID)
	}
	if pr.ProductID != uuid.Nil {
		t.Fatalf("expected zero ProductID, got %s", pr.ProductID)
	}
	if pr.AverageRating != 0 {
		t.Fatalf("expected AverageRating 0, got %f", pr.AverageRating)
	}
	if pr.ReviewCount != 0 {
		t.Fatalf("expected ReviewCount 0, got %d", pr.ReviewCount)
	}
	if !pr.UpdatedAt.IsZero() {
		t.Fatalf("expected zero UpdatedAt, got %v", pr.UpdatedAt)
	}
}

func TestUpdateReviewInput_PartialUpdate(t *testing.T) {
	t.Parallel()

	t.Run("all nil", func(t *testing.T) {
		input := domain.UpdateReviewInput{}

		if input.Rating != nil {
			t.Fatal("expected Rating to be nil")
		}
		if input.Title != nil {
			t.Fatal("expected Title to be nil")
		}
		if input.Body != nil {
			t.Fatal("expected Body to be nil")
		}
	})

	t.Run("only rating set", func(t *testing.T) {
		rating := 3
		input := domain.UpdateReviewInput{
			Rating: &rating,
		}

		if input.Rating == nil {
			t.Fatal("expected Rating to be non-nil")
		}
		if *input.Rating != 3 {
			t.Fatalf("expected Rating 3, got %d", *input.Rating)
		}
		if input.Title != nil {
			t.Fatal("expected Title to be nil")
		}
		if input.Body != nil {
			t.Fatal("expected Body to be nil")
		}
	})

	t.Run("title and body set", func(t *testing.T) {
		title := "Updated Title"
		body := "Updated Body"
		input := domain.UpdateReviewInput{
			Title: &title,
			Body:  &body,
		}

		if input.Rating != nil {
			t.Fatal("expected Rating to be nil")
		}
		if input.Title == nil || *input.Title != "Updated Title" {
			t.Fatalf("expected Title %q, got %v", "Updated Title", input.Title)
		}
		if input.Body == nil || *input.Body != "Updated Body" {
			t.Fatalf("expected Body %q, got %v", "Updated Body", input.Body)
		}
	})

	t.Run("all set", func(t *testing.T) {
		rating := 5
		title := "New Title"
		body := "New Body"
		input := domain.UpdateReviewInput{
			Rating: &rating,
			Title:  &title,
			Body:   &body,
		}

		if input.Rating == nil || *input.Rating != 5 {
			t.Fatalf("expected Rating 5, got %v", input.Rating)
		}
		if input.Title == nil || *input.Title != "New Title" {
			t.Fatalf("expected Title %q, got %v", "New Title", input.Title)
		}
		if input.Body == nil || *input.Body != "New Body" {
			t.Fatalf("expected Body %q, got %v", "New Body", input.Body)
		}
	})
}

func TestCreateReviewInput_Fields(t *testing.T) {
	t.Parallel()

	productID := uuid.New()
	input := domain.CreateReviewInput{
		ProductID: productID,
		Rating:    5,
		Title:     "Amazing Product",
		Body:      "This product exceeded my expectations.",
	}

	if input.ProductID != productID {
		t.Fatalf("expected ProductID %s, got %s", productID, input.ProductID)
	}
	if input.Rating != 5 {
		t.Fatalf("expected Rating 5, got %d", input.Rating)
	}
	if input.Title != "Amazing Product" {
		t.Fatalf("expected Title %q, got %q", "Amazing Product", input.Title)
	}
	if input.Body != "This product exceeded my expectations." {
		t.Fatalf("expected Body %q, got %q", "This product exceeded my expectations.", input.Body)
	}
}
