package app

import (
	"context"
	"testing"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// TestApplyCancellation_RefundAndReverse pins the ledger math for a
// canceled paid order: the redeemed points return and the earned
// points unwind, all in one transaction with a consistent running
// balance.
func TestApplyCancellation_RefundAndReverse(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{
			BuyerAuth0ID:   "auth0|buyer",
			Balance:        200, // after paying: 1000 redeemed, 100 earned, 200 = arbitrary post-paid balance
			LifetimeEarned: 100,
			Version:        2,
		},
	}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 0, 0)

	out, err := svc.ApplyCancellation(context.Background(), port.CancelPointsInput{
		BuyerAuth0ID:        "auth0|buyer",
		OrderID:             "order-1",
		PointDiscountAmount: 1000,
		PointsEarned:        100,
	})
	if err != nil {
		t.Fatalf("ApplyCancellation() error = %v", err)
	}
	if out.Refunded != 1000 {
		t.Errorf("refunded = %d, want 1000", out.Refunded)
	}
	if out.Reversed != 100 {
		t.Errorf("reversed = %d, want 100", out.Reversed)
	}
	// Balance: 200 + 1000 (refund) - 100 (reverse) = 1100
	if accounts.acct.Balance != 1100 {
		t.Errorf("final balance = %d, want 1100", accounts.acct.Balance)
	}
	// Two ledger rows should have been appended.
	if len(transactions.inserts) != 2 {
		t.Errorf("ledger rows written = %d, want 2", len(transactions.inserts))
	}
}

// TestApplyCancellation_IdempotentReplay pins the webhook-replay
// behavior: a second delivery for the same order must not re-refund
// or re-reverse — the unique idempotency key blocks the second insert.
func TestApplyCancellation_IdempotentReplay(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{
			BuyerAuth0ID:   "auth0|buyer",
			Balance:        1100,
			LifetimeEarned: 0, // already reversed
			Version:        4,
		},
	}
	// Pre-seed the ledger as if the first cancellation already landed.
	transactions := &fakeTransactionStore{
		byIdem: map[string]*domain.Transaction{
			idemKey(domain.SourceTypeOrderCancelled, "order-1", domain.TransactionTypeRefund):      {Amount: 1000},
			idemKey(domain.SourceTypeOrderCancelled, "order-1", domain.TransactionTypeReverseEarn): {Amount: -100},
		},
	}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 0, 0)

	out, err := svc.ApplyCancellation(context.Background(), port.CancelPointsInput{
		BuyerAuth0ID:        "auth0|buyer",
		OrderID:             "order-1",
		PointDiscountAmount: 1000,
		PointsEarned:        100,
	})
	if err != nil {
		t.Fatalf("ApplyCancellation() error = %v", err)
	}
	if out.Refunded != 0 || out.Reversed != 0 {
		t.Errorf("expected no-op on replay, got refunded=%d reversed=%d", out.Refunded, out.Reversed)
	}
	// Balance must not change.
	if accounts.acct.Balance != 1100 {
		t.Errorf("balance changed on replay: got %d, want 1100", accounts.acct.Balance)
	}
}

// TestApplyCancellation_ReverseCapByBalance covers the guard against
// driving the ledger negative: a buyer who spent their earn on
// another purchase has a smaller balance than the requested
// reverse_earn — the reversal caps at balance.
func TestApplyCancellation_ReverseCapByBalance(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{
			BuyerAuth0ID:   "auth0|buyer",
			Balance:        30, // buyer spent most of their balance
			LifetimeEarned: 100,
			Version:        3,
		},
	}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 0, 0)

	out, err := svc.ApplyCancellation(context.Background(), port.CancelPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		OrderID:      "order-1",
		PointsEarned: 100, // wanted to reverse 100 but only 30 available
	})
	if err != nil {
		t.Fatalf("ApplyCancellation() error = %v", err)
	}
	if out.Reversed != 30 {
		t.Errorf("reversed = %d, want 30 (capped by balance)", out.Reversed)
	}
	// Balance must hit 0 cleanly — never go negative.
	if accounts.acct.Balance != 0 {
		t.Errorf("final balance = %d, want 0", accounts.acct.Balance)
	}
}

// TestApplyCancellation_NoOpWhenAmountsZero: an order that had no
// discount and no earn (e.g. zero earn rate) should short-circuit
// without any DB calls.
func TestApplyCancellation_NoOpWhenAmountsZero(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 0, 0)

	out, err := svc.ApplyCancellation(context.Background(), port.CancelPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		OrderID:      "order-1",
	})
	if err != nil {
		t.Fatalf("ApplyCancellation() error = %v", err)
	}
	if out.Refunded != 0 || out.Reversed != 0 {
		t.Errorf("expected zero reversal, got refunded=%d reversed=%d", out.Refunded, out.Reversed)
	}
	if len(accounts.calls) != 0 {
		t.Errorf("expected no account calls, got %v", accounts.calls)
	}
}
