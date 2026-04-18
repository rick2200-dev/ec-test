package domain

import (
	"testing"
	"time"
)

func intPtr(n int) *int     { return &n }
func int64Ptr(n int64) *int64 { return &n }
func timePtr(t time.Time) *time.Time { return &t }

// TestComputeDiscount_Percent pins the basis-points math and the
// max-cap behavior. Every row is (name, coupon, subtotal, want).
func TestComputeDiscount_Percent(t *testing.T) {
	cases := []struct {
		name     string
		coupon   Coupon
		subtotal int64
		want     int64
	}{
		{
			name:     "10% of 1000 yen is 100",
			coupon:   Coupon{DiscountType: DiscountTypePercent, DiscountPercentBps: intPtr(1000)},
			subtotal: 1000,
			want:     100,
		},
		{
			name:     "10% of 1999 yen floors to 199",
			coupon:   Coupon{DiscountType: DiscountTypePercent, DiscountPercentBps: intPtr(1000)},
			subtotal: 1999,
			want:     199,
		},
		{
			name:     "10% capped by max_discount_amount",
			coupon:   Coupon{DiscountType: DiscountTypePercent, DiscountPercentBps: intPtr(1000), MaxDiscountAmount: int64Ptr(150)},
			subtotal: 5000,
			want:     150,
		},
		{
			name:     "50% of 200 is 100 (below cap)",
			coupon:   Coupon{DiscountType: DiscountTypePercent, DiscountPercentBps: intPtr(5000), MaxDiscountAmount: int64Ptr(500)},
			subtotal: 200,
			want:     100,
		},
		{
			name:     "zero subtotal earns nothing",
			coupon:   Coupon{DiscountType: DiscountTypePercent, DiscountPercentBps: intPtr(1000)},
			subtotal: 0,
			want:     0,
		},
		{
			name:     "negative subtotal is clamped to 0",
			coupon:   Coupon{DiscountType: DiscountTypePercent, DiscountPercentBps: intPtr(1000)},
			subtotal: -500,
			want:     0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.coupon.ComputeDiscount(tc.subtotal); got != tc.want {
				t.Errorf("ComputeDiscount(%d) = %d, want %d", tc.subtotal, got, tc.want)
			}
		})
	}
}

// TestComputeDiscount_FixedAmount covers the "capped to subtotal" path
// so a 500-yen coupon on a 300-yen cart still doesn't go negative.
func TestComputeDiscount_FixedAmount(t *testing.T) {
	coupon := Coupon{DiscountType: DiscountTypeFixedAmount, DiscountAmount: int64Ptr(500)}
	if got := coupon.ComputeDiscount(1000); got != 500 {
		t.Errorf("ComputeDiscount(1000) = %d, want 500", got)
	}
	if got := coupon.ComputeDiscount(300); got != 300 {
		t.Errorf("ComputeDiscount(300) = %d, want 300 (capped by subtotal)", got)
	}
}

// TestIsWithinValidity covers the three windows: before valid_from,
// inside, and after expires_at. Reserve uses this to reject expired
// coupons even when status is still 'active'.
func TestIsWithinValidity(t *testing.T) {
	base := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	coupon := Coupon{
		ValidFrom: base,
		ExpiresAt: timePtr(base.Add(24 * time.Hour)),
	}

	if coupon.IsWithinValidity(base.Add(-time.Hour)) {
		t.Error("should be invalid before valid_from")
	}
	if !coupon.IsWithinValidity(base.Add(time.Hour)) {
		t.Error("should be valid within window")
	}
	if coupon.IsWithinValidity(base.Add(48 * time.Hour)) {
		t.Error("should be invalid after expires_at")
	}

	// Nil expires_at = open-ended.
	openEnded := Coupon{ValidFrom: base}
	if !openEnded.IsWithinValidity(base.Add(1000 * time.Hour)) {
		t.Error("open-ended coupon should remain valid far in the future")
	}
}
