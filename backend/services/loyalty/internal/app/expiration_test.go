package app

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
)

// fakeClock is a deterministic Clock for expiration tests. Direct
// struct (not a function seam) so sub-tests can advance time by
// assigning a new value.
type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time { return f.now }

// newExpirationFixture assembles a Service wired with an in-memory
// store and a fake clock. 365d runway is the product default; tests
// pre-seed the ledger via direct store.Insert so the service never
// actually writes expiration-bearing earns via its own API — that's
// exercised separately in TestAwardEarn_StampsExpiresAt.
func newExpirationFixture(t *testing.T) (*Service, *fakeAccountStore, *fakeTransactionStore, *fakeClock) {
	t.Helper()
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	clk := &fakeClock{now: time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 365*24*time.Hour, 200)
	svc.SetClock(clk)
	return svc, accounts, transactions, clk
}

// seedEarn appends an earn row to the in-memory store without going
// through AwardEarn, so tests can place the earn arbitrarily far in
// the past (AwardEarn's clock is the same fakeClock, so it couldn't
// produce a pre-dated earn in a single test).
func seedEarn(store *fakeTransactionStore, buyer string, amount int64, earnedAt time.Time, expiresAt time.Time, orderID string) *domain.Transaction {
	t := &domain.Transaction{
		ID:           uuid.New(),
		BuyerAuth0ID: buyer,
		Type:         domain.TransactionTypeEarn,
		Amount:       amount,
		BalanceAfter: 0, // tests don't assert on historical balance_after
		SourceType:   domain.SourceTypeOrderPaid,
		SourceID:     orderID,
		CreatedAt:    earnedAt,
		ExpiresAt:    &expiresAt,
	}
	if store.byIdem == nil {
		store.byIdem = map[string]*domain.Transaction{}
	}
	store.byIdem[idemKey(t.SourceType, t.SourceID, t.Type)] = t
	store.inserts = append(store.inserts, t)
	return t
}

func seedRedeem(store *fakeTransactionStore, buyer string, amount int64, at time.Time, reservationID uuid.UUID) *domain.Transaction {
	t := &domain.Transaction{
		ID:            uuid.New(),
		BuyerAuth0ID:  buyer,
		Type:          domain.TransactionTypeRedeem,
		Amount:        -amount,
		SourceType:    domain.SourceTypeReservationCommit,
		SourceID:      reservationID.String(),
		CreatedAt:     at,
		ReservationID: &reservationID,
	}
	if store.byIdem == nil {
		store.byIdem = map[string]*domain.Transaction{}
	}
	store.byIdem[idemKey(t.SourceType, t.SourceID, t.Type)] = t
	store.inserts = append(store.inserts, t)
	return t
}

// TestExpirePoints_Basic: a single expired earn row gets retired in
// one pass, balance decrements, and the expire ledger row carries the
// correct idempotency key so a re-run is a no-op.
func TestExpirePoints_Basic(t *testing.T) {
	svc, accounts, transactions, clk := newExpirationFixture(t)
	accounts.acct = &domain.Account{
		BuyerAuth0ID:   "auth0|buyer",
		Balance:        1000,
		LifetimeEarned: 1000,
		Version:        1,
	}
	earn := seedEarn(transactions, "auth0|buyer", 1000,
		clk.now.Add(-400*24*time.Hour),
		clk.now.Add(-1*24*time.Hour), // expired yesterday
		"order-A")

	n, err := svc.ExpirePoints(context.Background())
	if err != nil {
		t.Fatalf("ExpirePoints() error = %v", err)
	}
	if n != 1 {
		t.Errorf("expired buckets = %d, want 1", n)
	}
	if accounts.acct.Balance != 0 {
		t.Errorf("balance = %d, want 0 after expiring 1000", accounts.acct.Balance)
	}
	// The written expire row should carry source_id = earn.ID.String() so
	// a re-run short-circuits on the UNIQUE index.
	found := false
	for _, row := range transactions.inserts {
		if row.Type != domain.TransactionTypeExpire {
			continue
		}
		if row.SourceID != earn.ID.String() {
			t.Errorf("expire.source_id = %s, want %s", row.SourceID, earn.ID.String())
		}
		if row.Amount != -1000 {
			t.Errorf("expire.amount = %d, want -1000", row.Amount)
		}
		found = true
	}
	if !found {
		t.Fatal("no expire row was written")
	}
}

// TestExpirePoints_PartiallyConsumed: an earn already partly spent by
// a subsequent redeem should only expire the remainder, not the
// original face value.
func TestExpirePoints_PartiallyConsumed(t *testing.T) {
	svc, accounts, transactions, clk := newExpirationFixture(t)
	accounts.acct = &domain.Account{
		BuyerAuth0ID:     "auth0|buyer",
		Balance:          700, // 1000 earned, 300 redeemed
		LifetimeEarned:   1000,
		LifetimeRedeemed: 300,
		Version:          1,
	}
	seedEarn(transactions, "auth0|buyer", 1000,
		clk.now.Add(-400*24*time.Hour),
		clk.now.Add(-1*24*time.Hour),
		"order-A")
	seedRedeem(transactions, "auth0|buyer", 300,
		clk.now.Add(-100*24*time.Hour), uuid.New())

	n, err := svc.ExpirePoints(context.Background())
	if err != nil {
		t.Fatalf("ExpirePoints() error = %v", err)
	}
	if n != 1 {
		t.Errorf("expired buckets = %d, want 1", n)
	}
	if accounts.acct.Balance != 0 {
		t.Errorf("balance = %d, want 0 (700 remaining expired)", accounts.acct.Balance)
	}
	for _, row := range transactions.inserts {
		if row.Type == domain.TransactionTypeExpire {
			if row.Amount != -700 {
				t.Errorf("expire.amount = %d, want -700 (partial-consume)", row.Amount)
			}
		}
	}
}

// TestExpirePoints_IdempotentReRun: a second call on the same state
// must write zero new rows and zero new deltas.
func TestExpirePoints_IdempotentReRun(t *testing.T) {
	svc, accounts, transactions, clk := newExpirationFixture(t)
	accounts.acct = &domain.Account{
		BuyerAuth0ID:   "auth0|buyer",
		Balance:        500,
		LifetimeEarned: 500,
		Version:        1,
	}
	seedEarn(transactions, "auth0|buyer", 500,
		clk.now.Add(-400*24*time.Hour),
		clk.now.Add(-1*24*time.Hour),
		"order-A")

	if _, err := svc.ExpirePoints(context.Background()); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	balanceAfterFirst := accounts.acct.Balance
	insertsAfterFirst := len(transactions.inserts)

	if _, err := svc.ExpirePoints(context.Background()); err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if accounts.acct.Balance != balanceAfterFirst {
		t.Errorf("balance mutated on re-run: before=%d after=%d", balanceAfterFirst, accounts.acct.Balance)
	}
	if len(transactions.inserts) != insertsAfterFirst {
		t.Errorf("new rows written on re-run: before=%d after=%d", insertsAfterFirst, len(transactions.inserts))
	}
}

// TestExpirePoints_MultipleBucketsFIFO: with two earns in flight and
// a redeem that drains the older one, only the NEW earn (not expired
// yet) carries remaining balance — the old bucket is empty so nothing
// gets expired even though its expires_at is past.
func TestExpirePoints_MultipleBucketsFIFO(t *testing.T) {
	svc, accounts, transactions, clk := newExpirationFixture(t)
	accounts.acct = &domain.Account{
		BuyerAuth0ID:     "auth0|buyer",
		Balance:          1500, // 1000 + 1000 - 500 redeemed
		LifetimeEarned:   2000,
		LifetimeRedeemed: 500,
		Version:          1,
	}
	// Oldest earn: expires yesterday. 1000 pts.
	seedEarn(transactions, "auth0|buyer", 1000,
		clk.now.Add(-400*24*time.Hour),
		clk.now.Add(-1*24*time.Hour),
		"order-A")
	// Newer earn: expires in a year. 1000 pts.
	seedEarn(transactions, "auth0|buyer", 1000,
		clk.now.Add(-30*24*time.Hour),
		clk.now.Add(335*24*time.Hour),
		"order-B")
	// Redeem 500: FIFO-consumes half of the OLDER bucket.
	seedRedeem(transactions, "auth0|buyer", 500,
		clk.now.Add(-10*24*time.Hour), uuid.New())

	n, err := svc.ExpirePoints(context.Background())
	if err != nil {
		t.Fatalf("ExpirePoints() error = %v", err)
	}
	if n != 1 {
		t.Errorf("expired buckets = %d, want 1 (only the old bucket's remaining 500)", n)
	}
	if accounts.acct.Balance != 1000 {
		t.Errorf("balance = %d, want 1000 (1500 - 500 expired)", accounts.acct.Balance)
	}
	for _, row := range transactions.inserts {
		if row.Type == domain.TransactionTypeExpire {
			if row.Amount != -500 {
				t.Errorf("expire.amount = %d, want -500", row.Amount)
			}
		}
	}
}

// TestExpirePoints_NoExpiredEarns is the fast path: no candidates →
// zero work, nothing written, no lock taken.
func TestExpirePoints_NoExpiredEarns(t *testing.T) {
	svc, accounts, transactions, clk := newExpirationFixture(t)
	accounts.acct = &domain.Account{
		BuyerAuth0ID: "auth0|buyer", Balance: 1000, LifetimeEarned: 1000, Version: 1,
	}
	seedEarn(transactions, "auth0|buyer", 1000,
		clk.now.Add(-30*24*time.Hour),
		clk.now.Add(335*24*time.Hour), // not expired yet
		"order-A")

	n, err := svc.ExpirePoints(context.Background())
	if err != nil {
		t.Fatalf("ExpirePoints() error = %v", err)
	}
	if n != 0 {
		t.Errorf("expired buckets = %d, want 0", n)
	}
	if indexOf(accounts.calls, "LockForUpdate") >= 0 {
		t.Errorf("LockForUpdate was called on empty-candidate path: %v", accounts.calls)
	}
}

// TestExpirePoints_InvariantMismatch: corrupted account balance (the
// ledger sums to 500 but the account row says 900) is caught by the
// invariant check and the pass aborts without writing expire rows or
// mutating the account.
func TestExpirePoints_InvariantMismatch(t *testing.T) {
	svc, accounts, transactions, clk := newExpirationFixture(t)
	accounts.acct = &domain.Account{
		BuyerAuth0ID:   "auth0|buyer",
		Balance:        900, // ledger says 500; deliberate inconsistency
		LifetimeEarned: 500,
		Version:        1,
	}
	seedEarn(transactions, "auth0|buyer", 500,
		clk.now.Add(-400*24*time.Hour),
		clk.now.Add(-1*24*time.Hour),
		"order-A")
	startingInserts := len(transactions.inserts)

	n, err := svc.ExpirePoints(context.Background())
	if err != nil {
		t.Fatalf("ExpirePoints() error = %v (expected nil — errors are per-buyer and logged, not returned)", err)
	}
	if n != 0 {
		t.Errorf("expired buckets = %d, want 0 on invariant failure", n)
	}
	if accounts.acct.Balance != 900 {
		t.Errorf("account balance mutated on invariant failure: got %d, want 900", accounts.acct.Balance)
	}
	if len(transactions.inserts) != startingInserts {
		t.Errorf("inserts written on invariant failure: started=%d now=%d", startingInserts, len(transactions.inserts))
	}
}

// TestAwardEarn_StampsExpiresAt is the mirror assertion for the
// write path: when earnExpiration > 0 on NewService, every new earn
// carries a populated ExpiresAt at now() + runway.
func TestAwardEarn_StampsExpiresAt(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	clk := &fakeClock{now: time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 365*24*time.Hour, 0)
	svc.SetClock(clk)

	row, err := svc.AwardEarn(context.Background(), "auth0|buyer", uuid.New().String(), 10000, "JPY")
	if err != nil {
		t.Fatalf("AwardEarn() error = %v", err)
	}
	if row == nil {
		t.Fatal("AwardEarn() returned nil row")
	}
	if row.ExpiresAt == nil {
		t.Fatal("earn row was written with ExpiresAt=nil, expected stamped")
	}
	want := clk.now.Add(365 * 24 * time.Hour)
	if !row.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %s, want %s", row.ExpiresAt, want)
	}
}

// TestAwardEarn_SkipsExpiresAtWhenDisabled: when NewService gets a
// zero earnExpiration, earns stay permanent (ExpiresAt nil). Keeps
// the feature backward compatible with pre-expiration deployments.
func TestAwardEarn_SkipsExpiresAtWhenDisabled(t *testing.T) {
	accounts := &fakeAccountStore{}
	transactions := &fakeTransactionStore{}
	svc := NewService(accounts, transactions, nil, nullTxRunner{}, nil, 100, 0, 0, 0)

	row, err := svc.AwardEarn(context.Background(), "auth0|buyer", uuid.New().String(), 10000, "JPY")
	if err != nil {
		t.Fatalf("AwardEarn() error = %v", err)
	}
	if row.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %s, want nil when earnExpiration=0", row.ExpiresAt)
	}
}
