package domain_test

import (
	"errors"
	"testing"

	domain "github.com/Riku-KANO/ec-test/services/catalog/internal/domain"
)

func TestProductStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status domain.ProductStatus
		want   string
	}{
		{"StatusDraft", domain.StatusDraft, "draft"},
		{"StatusActive", domain.StatusActive, "active"},
		{"StatusArchived", domain.StatusArchived, "archived"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("got %q, want %q", tt.status, tt.want)
			}
		})
	}
}

func TestProductFilter_Defaults(t *testing.T) {
	var f domain.ProductFilter

	if f.SellerID != nil {
		t.Error("expected SellerID to be nil")
	}
	if f.Status != nil {
		t.Error("expected Status to be nil")
	}
	if f.CategoryID != nil {
		t.Error("expected CategoryID to be nil")
	}
}

func TestProductWithSKUs_EmptySKUs(t *testing.T) {
	p := domain.ProductWithSKUs{}

	if p.SKUs != nil {
		t.Error("expected SKUs to be nil for zero value")
	}
	if len(p.SKUs) != 0 {
		t.Errorf("expected len(SKUs) == 0, got %d", len(p.SKUs))
	}
}

func TestDomainErrors_NotNil(t *testing.T) {
	errs := []struct {
		name string
		err  error
	}{
		{"ErrCategoryNotFound", domain.ErrCategoryNotFound},
		{"ErrCategorySlugConflict", domain.ErrCategorySlugConflict},
		{"ErrProductNotFound", domain.ErrProductNotFound},
		{"ErrProductSlugConflict", domain.ErrProductSlugConflict},
		{"ErrSKUNotFound", domain.ErrSKUNotFound},
		{"ErrSellerRequired", domain.ErrSellerRequired},
		{"ErrNotProductOwner", domain.ErrNotProductOwner},
	}
	for _, tt := range errs {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

func TestDomainErrors_Distinct(t *testing.T) {
	errs := []error{
		domain.ErrCategoryNotFound,
		domain.ErrCategorySlugConflict,
		domain.ErrProductNotFound,
		domain.ErrProductSlugConflict,
		domain.ErrSKUNotFound,
		domain.ErrSellerRequired,
		domain.ErrNotProductOwner,
	}

	seen := make(map[string]bool, len(errs))
	for _, err := range errs {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}

func TestDomainErrors_Unwrap(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrCategoryNotFound", domain.ErrCategoryNotFound},
		{"ErrCategorySlugConflict", domain.ErrCategorySlugConflict},
		{"ErrProductNotFound", domain.ErrProductNotFound},
		{"ErrProductSlugConflict", domain.ErrProductSlugConflict},
		{"ErrSKUNotFound", domain.ErrSKUNotFound},
		{"ErrSellerRequired", domain.ErrSellerRequired},
		{"ErrNotProductOwner", domain.ErrNotProductOwner},
	}

	for _, tt := range sentinels {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.err) {
				t.Errorf("errors.Is(%s, %s) should be true", tt.name, tt.name)
			}
		})
	}

	// Verify that distinct sentinels are not equal to each other.
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a.err, b.err) {
				t.Errorf("errors.Is(%s, %s) should be false", a.name, b.name)
			}
		}
	}
}
