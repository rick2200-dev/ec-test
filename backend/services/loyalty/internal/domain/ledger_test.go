package domain

import "testing"

// TestComputeEarnAmount pins the integer-only floor semantics so the
// accrual never introduces fractional yen via silent FP conversions.
// Each row is a (name, paidSubtotal, earnRateBps, expected) tuple.
func TestComputeEarnAmount(t *testing.T) {
	cases := []struct {
		name     string
		subtotal int64
		rateBps  int
		want     int64
	}{
		{"zero subtotal earns nothing", 0, 100, 0},
		{"negative subtotal earns nothing", -500, 100, 0},
		{"disabled rate earns nothing", 1000, 0, 0},
		{"1% of 1000 yen is 10", 1000, 100, 10},
		{"1% of 1999 yen floors to 19", 1999, 100, 19},
		{"5% of 2500 yen is 125", 2500, 500, 125},
		{"2.5% of 1000 yen floors to 25 via bps", 1000, 250, 25},
		{"1% of 99 yen floors to 0 (below 1 yen threshold)", 99, 100, 0},
		{"10% of 1 yen still floors to 0", 1, 1000, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeEarnAmount(tc.subtotal, tc.rateBps)
			if got != tc.want {
				t.Errorf("ComputeEarnAmount(%d, %d) = %d, want %d", tc.subtotal, tc.rateBps, got, tc.want)
			}
		})
	}
}

// TestAccountSpendable documents that spendable = balance - pending. The
// handlers and the gRPC response rely on this property when they return
// Balance to the buyer UI.
func TestAccountSpendable(t *testing.T) {
	a := Account{Balance: 1000, PendingRedemption: 300}
	if got := a.Spendable(); got != 700 {
		t.Errorf("Spendable() = %d, want 700", got)
	}

	empty := Account{}
	if got := empty.Spendable(); got != 0 {
		t.Errorf("Spendable() empty = %d, want 0", got)
	}
}
