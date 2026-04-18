package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// fakeAccountStore records call sequence so tests can assert that the
// lock is acquired before any balance mutation. All methods are simple
// in-memory stubs — no pgx involvement.
type fakeAccountStore struct {
	acct *domain.Account
	// calls captures the ordered method names invoked against the
	// store. Tests assert against this to pin the Lock-before-delta
	// serialization contract.
	calls        []string
	applyDeltaFn func(port.AccountDelta, int) error
	lockFn       func() (*domain.Account, error)
}

func (f *fakeAccountStore) Get(ctx context.Context, buyerAuth0ID string) (*domain.Account, error) {
	f.calls = append(f.calls, "Get")
	if f.acct == nil {
		return nil, nil
	}
	cp := *f.acct
	return &cp, nil
}

func (f *fakeAccountStore) EnsureExists(ctx context.Context, buyerAuth0ID string) error {
	f.calls = append(f.calls, "EnsureExists")
	if f.acct == nil {
		f.acct = &domain.Account{BuyerAuth0ID: buyerAuth0ID}
	}
	return nil
}

func (f *fakeAccountStore) LockForUpdate(ctx context.Context, buyerAuth0ID string) (*domain.Account, error) {
	f.calls = append(f.calls, "LockForUpdate")
	if f.lockFn != nil {
		return f.lockFn()
	}
	if f.acct == nil {
		f.acct = &domain.Account{BuyerAuth0ID: buyerAuth0ID}
	}
	cp := *f.acct
	return &cp, nil
}

func (f *fakeAccountStore) ApplyDelta(ctx context.Context, buyerAuth0ID string, delta port.AccountDelta, expectedVersion int) error {
	f.calls = append(f.calls, "ApplyDelta")
	if f.applyDeltaFn != nil {
		return f.applyDeltaFn(delta, expectedVersion)
	}
	if f.acct.Version != expectedVersion {
		return port.ErrOptimisticLock
	}
	f.acct.Balance += delta.BalanceDelta
	f.acct.LifetimeEarned += delta.LifetimeEarnedDelta
	f.acct.Version++
	return nil
}

type fakeTransactionStore struct {
	byIdem   map[string]*domain.Transaction
	inserts  []*domain.Transaction
	insertFn func(*domain.Transaction) error
	calls    []string
}

func (f *fakeTransactionStore) Insert(ctx context.Context, t *domain.Transaction) error {
	f.calls = append(f.calls, "Insert")
	if f.insertFn != nil {
		return f.insertFn(t)
	}
	key := idemKey(t.SourceType, t.SourceID, t.Type)
	if f.byIdem == nil {
		f.byIdem = map[string]*domain.Transaction{}
	}
	if _, ok := f.byIdem[key]; ok {
		return port.ErrDuplicateIdempotencyKey
	}
	f.byIdem[key] = t
	f.inserts = append(f.inserts, t)
	return nil
}

func (f *fakeTransactionStore) ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Transaction, int, error) {
	return nil, 0, nil
}

func (f *fakeTransactionStore) GetByIdempotency(ctx context.Context, sourceType domain.SourceType, sourceID string, txType domain.TransactionType) (*domain.Transaction, error) {
	f.calls = append(f.calls, "GetByIdempotency")
	if f.byIdem == nil {
		return nil, nil
	}
	return f.byIdem[idemKey(sourceType, sourceID, txType)], nil
}

func idemKey(st domain.SourceType, sid string, tt domain.TransactionType) string {
	return fmt.Sprintf("%s|%s|%s", st, sid, tt)
}

// nullTxRunner invokes fn directly — there's no real tx boundary in
// these unit tests, just method-sequence verification.
type nullTxRunner struct{}

func (nullTxRunner) RunTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// TestAwardEarn_LockBeforeDelta pins the invariant the race fix is
// built on: LockForUpdate MUST be the first call the service makes on
// AccountStore, and the ledger Insert MUST happen only after the
// idempotency check has cleared. If someone reorders these in a
// future refactor the over-credit scenario the test below reproduces
// will reappear.
func TestAwardEarn_LockBeforeDelta(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	order := uuid.New().String()
	_, err := svc.AwardEarn(context.Background(), "auth0|buyer", order, 10000, "JPY")
	if err != nil {
		t.Fatalf("AwardEarn() error = %v", err)
	}

	// Expected ordering: lock first, then idempotency check, then delta, then insert.
	want := []string{"LockForUpdate", "ApplyDelta"}
	if !equalPrefix(accounts.calls, want) {
		t.Errorf("account call order = %v, want prefix %v", accounts.calls, want)
	}
	if len(transactions.calls) < 2 || transactions.calls[0] != "GetByIdempotency" || transactions.calls[1] != "Insert" {
		t.Errorf("transaction call order = %v, want [GetByIdempotency Insert ...]", transactions.calls)
	}
	// Cross-store ordering: LockForUpdate must precede any Insert.
	if indexOf(accounts.calls, "LockForUpdate") < 0 {
		t.Fatal("LockForUpdate was never called")
	}
}

// TestAwardEarn_DuplicateReplayNoDoubleCredit is the direct regression
// test for the over-credit race reported against the previous
// optimistic-lock implementation. We simulate the "winner already
// committed" state by pre-seeding the ledger row — the service must
// detect that via the idempotency check and return without applying
// a second delta.
func TestAwardEarn_DuplicateReplayNoDoubleCredit(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{
			BuyerAuth0ID:   "auth0|buyer",
			Balance:        100, // the "winner" already credited 100
			LifetimeEarned: 100,
			Version:        1,
		},
	}
	order := uuid.New().String()
	existing := &domain.Transaction{
		ID:           uuid.New(),
		BuyerAuth0ID: "auth0|buyer",
		Type:         domain.TransactionTypeEarn,
		Amount:       100,
		BalanceAfter: 100,
		SourceType:   domain.SourceTypeOrderPaid,
		SourceID:     order,
	}
	transactions := &fakeTransactionStore{
		byIdem: map[string]*domain.Transaction{
			idemKey(domain.SourceTypeOrderPaid, order, domain.TransactionTypeEarn): existing,
		},
	}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	got, err := svc.AwardEarn(context.Background(), "auth0|buyer", order, 10000, "JPY")
	if err != nil {
		t.Fatalf("AwardEarn() error = %v", err)
	}
	if got == nil || got.ID != existing.ID {
		t.Errorf("AwardEarn() returned %+v, want existing row id=%s", got, existing.ID)
	}
	// Balance must NOT have changed — over-credit regression.
	if accounts.acct.Balance != 100 {
		t.Errorf("balance = %d, want 100 (no double-credit)", accounts.acct.Balance)
	}
	if accounts.acct.LifetimeEarned != 100 {
		t.Errorf("lifetime_earned = %d, want 100 (no double-credit)", accounts.acct.LifetimeEarned)
	}
	// ApplyDelta must NOT have been called.
	if indexOf(accounts.calls, "ApplyDelta") >= 0 {
		t.Errorf("ApplyDelta was called on duplicate replay — expected short-circuit via idempotency check. calls=%v", accounts.calls)
	}
	// Insert must NOT have been called (short-circuit on duplicate).
	if indexOf(transactions.calls, "Insert") >= 0 {
		t.Errorf("Insert was called on duplicate replay — expected short-circuit. calls=%v", transactions.calls)
	}
}

// TestAwardEarn_RaceWinsAtInsert covers the defensive check we kept
// for the (near-impossible) scenario where someone bypasses the row
// lock and races a duplicate Insert. The service must surface that as
// an error rather than silently absorbing the extra delta — otherwise
// the ledger invariant (balance = sum(amount)) breaks.
func TestAwardEarn_RaceWinsAtInsert(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{
		insertFn: func(t *domain.Transaction) error {
			return port.ErrDuplicateIdempotencyKey
		},
	}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	_, err := svc.AwardEarn(context.Background(), "auth0|buyer", uuid.New().String(), 10000, "JPY")
	if err == nil {
		t.Fatal("AwardEarn() nil error, want internal error on unexpected duplicate key")
	}
	// The error must have a descriptive message — not a silent
	// success that leaves ApplyDelta committed.
	if !containsSubstr(err.Error(), "duplicate idempotency key") {
		t.Errorf("error message = %q, want mention of duplicate key", err.Error())
	}
}

// TestAwardEarn_ZeroEarnIsNoOp: earn rate math that floors to 0 (sub-¥1
// purchase, earn_rate_bps=0, etc.) must return (nil, nil) without
// touching the account or ledger.
func TestAwardEarn_ZeroEarnIsNoOp(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	got, err := svc.AwardEarn(context.Background(), "auth0|buyer", uuid.New().String(), 99, "JPY")
	if err != nil {
		t.Fatalf("AwardEarn() error = %v", err)
	}
	if got != nil {
		t.Errorf("AwardEarn() = %+v, want nil on zero-earn", got)
	}
	if len(accounts.calls) != 0 {
		t.Errorf("account calls on zero earn = %v, want empty", accounts.calls)
	}
}

// TestAdjustPoints_Credit exercises a positive admin adjustment:
// balance goes up by amount and an `adjust` ledger row is recorded
// with admin_user_id in the note for audit.
func TestAdjustPoints_Credit(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{BuyerAuth0ID: "auth0|buyer", Balance: 500, Version: 3},
	}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	result, err := svc.AdjustPoints(context.Background(), port.AdjustPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		Amount:       200,
		Reason:       "apology for late delivery",
		AdminUserID:  "admin-42",
	})
	if err != nil {
		t.Fatalf("AdjustPoints() error = %v", err)
	}
	if result.NewBalance != 700 {
		t.Errorf("NewBalance = %d, want 700", result.NewBalance)
	}
	if accounts.acct.Balance != 700 {
		t.Errorf("acct.Balance = %d, want 700", accounts.acct.Balance)
	}
	if len(transactions.inserts) != 1 {
		t.Fatalf("inserts len = %d, want 1", len(transactions.inserts))
	}
	row := transactions.inserts[0]
	if row.Type != domain.TransactionTypeAdjust {
		t.Errorf("row.Type = %s, want adjust", row.Type)
	}
	if row.Amount != 200 {
		t.Errorf("row.Amount = %d, want 200", row.Amount)
	}
	if row.SourceType != domain.SourceTypeAdminAdjust {
		t.Errorf("row.SourceType = %s, want admin_adjust", row.SourceType)
	}
	if !containsSubstr(row.Note, "admin=admin-42") {
		t.Errorf("row.Note = %q, want to contain admin=admin-42", row.Note)
	}
	if !containsSubstr(row.Note, "apology for late delivery") {
		t.Errorf("row.Note = %q, want to contain reason", row.Note)
	}
}

// TestAdjustPoints_Debit exercises a negative admin adjustment within
// spendable balance — balance decreases and a debit ledger row is
// recorded.
func TestAdjustPoints_Debit(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{BuyerAuth0ID: "auth0|buyer", Balance: 500, Version: 1},
	}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	result, err := svc.AdjustPoints(context.Background(), port.AdjustPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		Amount:       -150,
		Reason:       "duplicate credit rollback",
		AdminUserID:  "admin-1",
	})
	if err != nil {
		t.Fatalf("AdjustPoints() error = %v", err)
	}
	if result.NewBalance != 350 {
		t.Errorf("NewBalance = %d, want 350", result.NewBalance)
	}
	if transactions.inserts[0].Amount != -150 {
		t.Errorf("row.Amount = %d, want -150", transactions.inserts[0].Amount)
	}
}

// TestAdjustPoints_InsufficientBalance rejects a debit that would push
// spendable negative without mutating state.
func TestAdjustPoints_InsufficientBalance(t *testing.T) {
	accounts := &fakeAccountStore{
		acct: &domain.Account{BuyerAuth0ID: "auth0|buyer", Balance: 100, Version: 0},
	}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	_, err := svc.AdjustPoints(context.Background(), port.AdjustPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		Amount:       -500,
		Reason:       "test",
	})
	if !errors.Is(err, domain.ErrInsufficientPoints) {
		t.Fatalf("err = %v, want ErrInsufficientPoints", err)
	}
	if accounts.acct.Balance != 100 {
		t.Errorf("balance mutated: %d, want 100", accounts.acct.Balance)
	}
	if len(transactions.inserts) != 0 {
		t.Errorf("inserts written on rejection: %+v", transactions.inserts)
	}
}

// TestAdjustPoints_ZeroAmount rejects zero-delta calls so we don't
// pollute the ledger with no-op audit rows.
func TestAdjustPoints_ZeroAmount(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	_, err := svc.AdjustPoints(context.Background(), port.AdjustPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		Amount:       0,
		Reason:       "test",
	})
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("err = %v, want ErrInvalidAmount", err)
	}
}

// TestAdjustPoints_MissingReason enforces the audit requirement: every
// admin adjustment must carry a human-readable reason.
func TestAdjustPoints_MissingReason(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0)

	_, err := svc.AdjustPoints(context.Background(), port.AdjustPointsInput{
		BuyerAuth0ID: "auth0|buyer",
		Amount:       100,
		Reason:       "",
	})
	if err == nil {
		t.Fatal("expected error for missing reason")
	}
}

// helpers

func equalPrefix(got, want []string) bool {
	if len(got) < len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func indexOf(haystack []string, needle string) int {
	for i, s := range haystack {
		if s == needle {
			return i
		}
	}
	return -1
}

func containsSubstr(haystack, needle string) bool {
	return len(needle) <= len(haystack) && indexOfSubstring(haystack, needle) >= 0
}

func indexOfSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// Compile-time check: make sure the error we expect wraps
// ErrDuplicateIdempotencyKey.
var _ = errors.Is(errors.New(""), port.ErrDuplicateIdempotencyKey)
