package domain

import "github.com/google/uuid"

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

// DistributeSellerDiscount routes a seller-scoped discount to only the
// orders whose seller matches targetSellerID. When several orders in
// the batch belong to the target seller (a seller can occupy multiple
// batch slots if the cart merges differently-priced SKUs from the same
// seller), the discount is split across those slots proportionally to
// their subtotals using the same floor+last-absorbs-remainder math as
// DistributeDiscount. Non-matching slots get zero.
//
// sellerIDs and subtotals are 1:1 with the output slice — same length,
// same indexing.
//
// Seller-issued coupons apply only to the issuing seller's subtotal,
// so a cross-seller cart must not leak discount to other sellers'
// orders (which would misattribute refunds on cancellation and
// distort platform commission math).
func DistributeSellerDiscount(discount int64, sellerIDs []uuid.UUID, subtotals []int64, targetSellerID uuid.UUID) []int64 {
	if len(sellerIDs) != len(subtotals) {
		// Caller bug — return all zeros rather than panicking in prod.
		return make([]int64, len(subtotals))
	}
	// Build a reduced view containing only the matching seller's slots,
	// run the proportional split on that, then splice the result back
	// into the full output slice keyed by original index.
	matchIndex := make([]int, 0, len(sellerIDs))
	matchSubtotals := make([]int64, 0, len(sellerIDs))
	for i, sid := range sellerIDs {
		if sid == targetSellerID {
			matchIndex = append(matchIndex, i)
			matchSubtotals = append(matchSubtotals, subtotals[i])
		}
	}
	matchedShares := DistributeDiscount(discount, matchSubtotals)
	out := make([]int64, len(subtotals))
	for k, idx := range matchIndex {
		out[idx] = matchedShares[k]
	}
	return out
}
