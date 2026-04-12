package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/cart/internal/domain"
)

func newItem(price int64, qty int) domain.CartItem {
	return domain.CartItem{
		SKUID:             uuid.New(),
		SellerID:          uuid.New(),
		Quantity:          qty,
		UnitPriceSnapshot: price,
		Currency:          "jpy",
		AddedAt:           time.Now().UTC(),
	}
}

// --- Total ---

func TestCart_Total_Empty(t *testing.T) {
	c := &domain.Cart{}
	if got := c.Total(); got != 0 {
		t.Errorf("Total() = %d, want 0", got)
	}
}

func TestCart_Total_SingleItem(t *testing.T) {
	c := &domain.Cart{Items: []domain.CartItem{newItem(1000, 3)}}
	if got := c.Total(); got != 3000 {
		t.Errorf("Total() = %d, want 3000", got)
	}
}

func TestCart_Total_MultipleItems(t *testing.T) {
	c := &domain.Cart{
		Items: []domain.CartItem{
			newItem(1000, 2), // 2000
			newItem(500, 5),  // 2500
			newItem(200, 1),  // 200
		},
	}
	if got := c.Total(); got != 4700 {
		t.Errorf("Total() = %d, want 4700", got)
	}
}

func TestCart_Total_ZeroQuantity(t *testing.T) {
	c := &domain.Cart{Items: []domain.CartItem{newItem(1000, 0)}}
	if got := c.Total(); got != 0 {
		t.Errorf("Total() = %d, want 0", got)
	}
}

// --- IsEmpty ---

func TestCart_IsEmpty_True(t *testing.T) {
	c := &domain.Cart{}
	if !c.IsEmpty() {
		t.Error("IsEmpty() = false, want true")
	}
}

func TestCart_IsEmpty_EmptySlice(t *testing.T) {
	c := &domain.Cart{Items: []domain.CartItem{}}
	if !c.IsEmpty() {
		t.Error("IsEmpty() = false, want true for empty slice")
	}
}

func TestCart_IsEmpty_False(t *testing.T) {
	c := &domain.Cart{Items: []domain.CartItem{newItem(100, 1)}}
	if c.IsEmpty() {
		t.Error("IsEmpty() = true, want false")
	}
}

// --- FindItem ---

func TestCart_FindItem_Found(t *testing.T) {
	sku1 := uuid.New()
	sku2 := uuid.New()
	c := &domain.Cart{
		Items: []domain.CartItem{
			{SKUID: sku1, Quantity: 1},
			{SKUID: sku2, Quantity: 2},
		},
	}

	if got := c.FindItem(sku1); got != 0 {
		t.Errorf("FindItem(sku1) = %d, want 0", got)
	}
	if got := c.FindItem(sku2); got != 1 {
		t.Errorf("FindItem(sku2) = %d, want 1", got)
	}
}

func TestCart_FindItem_NotFound(t *testing.T) {
	c := &domain.Cart{
		Items: []domain.CartItem{
			{SKUID: uuid.New(), Quantity: 1},
		},
	}
	if got := c.FindItem(uuid.New()); got != -1 {
		t.Errorf("FindItem(missing) = %d, want -1", got)
	}
}

func TestCart_FindItem_EmptyCart(t *testing.T) {
	c := &domain.Cart{}
	if got := c.FindItem(uuid.New()); got != -1 {
		t.Errorf("FindItem on empty cart = %d, want -1", got)
	}
}
