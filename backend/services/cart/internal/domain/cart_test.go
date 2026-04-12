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

// --- AddItem ---

func TestCart_AddItem_NewItem(t *testing.T) {
	c := &domain.Cart{}
	item := newItem(500, 2)

	before := time.Now().UTC()
	c.AddItem(item)

	if len(c.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(c.Items))
	}
	if c.Items[0].SKUID != item.SKUID {
		t.Errorf("SKUID = %v, want %v", c.Items[0].SKUID, item.SKUID)
	}
	if c.Items[0].Quantity != 2 {
		t.Errorf("Quantity = %d, want 2", c.Items[0].Quantity)
	}
	if c.UpdatedAt.Before(before) {
		t.Error("UpdatedAt was not set")
	}
}

func TestCart_AddItem_ExistingSKU_MergesQuantity(t *testing.T) {
	item := newItem(300, 1)
	c := &domain.Cart{Items: []domain.CartItem{item}}

	dup := domain.CartItem{
		SKUID:             item.SKUID,
		SellerID:          item.SellerID,
		Quantity:          4,
		UnitPriceSnapshot: 300,
		Currency:          "jpy",
		AddedAt:           time.Now().UTC(),
	}
	c.AddItem(dup)

	if len(c.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1 (should merge, not append)", len(c.Items))
	}
	if c.Items[0].Quantity != 5 {
		t.Errorf("Quantity = %d, want 5 (1 + 4)", c.Items[0].Quantity)
	}
}

func TestCart_AddItem_MultipleAdds(t *testing.T) {
	c := &domain.Cart{}
	item1 := newItem(100, 1)
	item2 := newItem(200, 2)
	item3 := newItem(300, 3)

	c.AddItem(item1)
	c.AddItem(item2)
	c.AddItem(item3)

	if len(c.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(c.Items))
	}
	// Add again with same SKU as item1 — should merge, not grow the slice.
	merge := domain.CartItem{
		SKUID:             item1.SKUID,
		SellerID:          item1.SellerID,
		Quantity:          5,
		UnitPriceSnapshot: 100,
		Currency:          "jpy",
		AddedAt:           time.Now().UTC(),
	}
	c.AddItem(merge)

	if len(c.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3 after merge", len(c.Items))
	}
	if c.Items[0].Quantity != 6 {
		t.Errorf("Quantity = %d, want 6 (1 + 5)", c.Items[0].Quantity)
	}
}

// --- RemoveItem ---

func TestCart_RemoveItem_Existing(t *testing.T) {
	item1 := newItem(100, 1)
	item2 := newItem(200, 2)
	c := &domain.Cart{Items: []domain.CartItem{item1, item2}}

	before := time.Now().UTC()
	c.RemoveItem(item1.SKUID)

	if len(c.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(c.Items))
	}
	if c.Items[0].SKUID != item2.SKUID {
		t.Errorf("remaining SKUID = %v, want %v", c.Items[0].SKUID, item2.SKUID)
	}
	if c.UpdatedAt.Before(before) {
		t.Error("UpdatedAt was not set after remove")
	}
}

func TestCart_RemoveItem_NonExistent_Idempotent(t *testing.T) {
	item := newItem(100, 1)
	c := &domain.Cart{Items: []domain.CartItem{item}}
	original := c.UpdatedAt

	c.RemoveItem(uuid.New()) // SKU not in cart

	if len(c.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1 (no change)", len(c.Items))
	}
	if c.UpdatedAt != original {
		t.Error("UpdatedAt should not change when removing non-existent SKU")
	}
}

func TestCart_RemoveItem_EmptyCart(t *testing.T) {
	c := &domain.Cart{}
	original := c.UpdatedAt

	c.RemoveItem(uuid.New()) // no-op on empty cart

	if len(c.Items) != 0 {
		t.Fatalf("len(Items) = %d, want 0", len(c.Items))
	}
	if c.UpdatedAt != original {
		t.Error("UpdatedAt should not change on empty cart remove")
	}
}

// --- SetItemQuantity ---

func TestCart_SetItemQuantity_Success(t *testing.T) {
	item := newItem(500, 1)
	c := &domain.Cart{Items: []domain.CartItem{item}}

	before := time.Now().UTC()
	err := c.SetItemQuantity(item.SKUID, 10)

	if err != nil {
		t.Fatalf("SetItemQuantity() error = %v, want nil", err)
	}
	if c.Items[0].Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", c.Items[0].Quantity)
	}
	if c.UpdatedAt.Before(before) {
		t.Error("UpdatedAt was not set after SetItemQuantity")
	}
}

func TestCart_SetItemQuantity_ErrSKUNotInCart(t *testing.T) {
	item := newItem(500, 1)
	c := &domain.Cart{Items: []domain.CartItem{item}}

	err := c.SetItemQuantity(uuid.New(), 5)

	if err != domain.ErrSKUNotInCart {
		t.Errorf("SetItemQuantity() error = %v, want %v", err, domain.ErrSKUNotInCart)
	}
}

func TestCart_SetItemQuantity_EmptyCart(t *testing.T) {
	c := &domain.Cart{}

	err := c.SetItemQuantity(uuid.New(), 3)

	if err != domain.ErrSKUNotInCart {
		t.Errorf("SetItemQuantity() on empty cart error = %v, want %v", err, domain.ErrSKUNotInCart)
	}
}
