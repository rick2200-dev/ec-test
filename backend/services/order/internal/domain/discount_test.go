package domain

import (
	"testing"
)

// TestDistributeDiscount_Basic pins the proportional-split math that
// drives multi-seller discount allocation. Each case is (name,
// discount, subtotals, want). "last absorbs remainder" is the
// deterministic tie-break for rounding.
func TestDistributeDiscount_Basic(t *testing.T) {
	cases := []struct {
		name      string
		discount  int64
		subtotals []int64
		want      []int64
	}{
		{
			name:      "single seller takes the whole discount",
			discount:  200,
			subtotals: []int64{1000},
			want:      []int64{200},
		},
		{
			name:      "equal split of even discount",
			discount:  200,
			subtotals: []int64{1000, 1000},
			want:      []int64{100, 100},
		},
		{
			name:      "proportional to subtotal",
			discount:  300,
			subtotals: []int64{1000, 2000},
			want:      []int64{100, 200},
		},
		{
			name:      "remainder absorbed by last bucket",
			discount:  100,
			subtotals: []int64{333, 333, 334},
			want:      []int64{33, 33, 34},
		},
		{
			name:      "zero subtotals stay at zero",
			discount:  100,
			subtotals: []int64{0, 1000},
			want:      []int64{0, 100},
		},
		{
			name:      "share capped by subtotal",
			discount:  500,
			subtotals: []int64{100, 1000},
			want:      []int64{45, 455},
		},
		{
			name:      "zero discount returns zeros",
			discount:  0,
			subtotals: []int64{1000, 2000},
			want:      []int64{0, 0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DistributeDiscount(tc.discount, tc.subtotals)
			if len(got) != len(tc.want) {
				t.Fatalf("length mismatch: got %d, want %d", len(got), len(tc.want))
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %d, want %d (full got=%v)", i, got[i], tc.want[i], got)
				}
			}
			var total int64
			for _, v := range got {
				total += v
			}
			if total > tc.discount {
				t.Errorf("total %d exceeds requested discount %d", total, tc.discount)
			}
		})
	}
}
