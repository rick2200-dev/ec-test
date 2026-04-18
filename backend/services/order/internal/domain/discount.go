package domain

// DistributeDiscount splits a cart-wide discount across per-seller
// subtotals proportionally, using integer-only math and flooring so
// the total never exceeds `discount`. Any rounding remainder is
// absorbed by the last bucket, which keeps the sum of shares exactly
// equal to `discount` even when the raw proportional math would
// produce a fractional yen.
//
// subtotals is expected in the same order as the orders to be created
// so the returned slice lines up 1:1 (including the "last absorbs
// remainder" rule, which the caller relies on for determinism).
//
// The zero-subtotal case (entire cart is points-only, e.g.) returns
// zeros; callers are expected to gate on totalSubtotal > 0 before
// applying a percent discount — this helper stays defensive regardless.
func DistributeDiscount(discount int64, subtotals []int64) []int64 {
	out := make([]int64, len(subtotals))
	if discount <= 0 || len(subtotals) == 0 {
		return out
	}
	var total int64
	for _, s := range subtotals {
		if s > 0 {
			total += s
		}
	}
	if total <= 0 {
		return out
	}
	var assigned int64
	for i, s := range subtotals {
		if s <= 0 {
			continue
		}
		// Floor-divide to keep the sum under the requested discount; the
		// residual goes to the last non-zero bucket.
		share := discount * s / total
		out[i] = share
		assigned += share
	}
	remainder := discount - assigned
	if remainder > 0 {
		// Walk from the end so the last non-zero bucket absorbs the
		// rounding error deterministically.
		for i := len(subtotals) - 1; i >= 0; i-- {
			if subtotals[i] > 0 {
				out[i] += remainder
				break
			}
		}
	}
	// Cap each share by its subtotal — a share can't exceed the order it
	// applies to. In practice this only triggers when the caller asks
	// for a discount larger than the cart subtotal, which Reserve
	// prevents; the cap is cheap belt-and-suspenders.
	for i, s := range subtotals {
		if out[i] > s {
			out[i] = s
		}
	}
	return out
}
