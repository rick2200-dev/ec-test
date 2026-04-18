package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
)

// AccountStore abstracts point_accounts persistence. All methods honor
// the ambient DB transaction (pulled from context by the repo adapter).
type AccountStore interface {
	// Get returns the account for buyer, or (nil, nil) if none exists.
	Get(ctx context.Context, buyerAuth0ID string) (*domain.Account, error)

	// EnsureExists creates a zero-balance account row when missing.
	// Idempotent: safe to call on every earn/reserve.
	EnsureExists(ctx context.Context, buyerAuth0ID string) error

	// LockForUpdate acquires a row-level exclusive lock on the
	// buyer's account for the duration of the current DB
	// transaction and returns the locked snapshot. Must be called
	// inside a transaction; callers hold it until commit/rollback.
	//
	// This is the primary serialization primitive for earn / reserve
	// paths — with the row locked, concurrent calls block in Postgres
	// and the idempotency check performed afterwards is definitive:
	// no racing transaction can slip in an ApplyDelta before the
	// ledger insert, which is what the old optimistic-lock scheme
	// got wrong on duplicate Pub/Sub + sync delivery of order.paid.
	LockForUpdate(ctx context.Context, buyerAuth0ID string) (*domain.Account, error)

	// ApplyDelta atomically adjusts an account's balance / pending /
	// lifetime counters, bumping version. Fails with ErrOptimisticLock
	// if expectedVersion doesn't match.
	ApplyDelta(ctx context.Context, buyerAuth0ID string, delta AccountDelta, expectedVersion int) error
}

// AccountDelta is the change applied in a single ApplyDelta call. All
// fields are signed: positive credits, negative debits.
type AccountDelta struct {
	BalanceDelta           int64
	PendingRedemptionDelta int64
	LifetimeEarnedDelta    int64
	LifetimeRedeemedDelta  int64
}

// ReservationStore abstracts loyalty_svc.point_reservations.
type ReservationStore interface {
	Insert(ctx context.Context, r *domain.Reservation) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Reservation, error)

	// UpdateStatus transitions a reservation. Returns (false, nil)
	// when the row wasn't in `expect` — app translates to
	// ErrReservationAlreadyTerminal.
	UpdateStatus(ctx context.Context, id uuid.UUID, expect, next domain.ReservationStatus) (bool, error)

	// ClaimExpired flips at most `limit` pending rows past their TTL
	// to expired and returns (id, buyer, amount) tuples so the reaper
	// can decrement pending_redemption for each without a second
	// round-trip.
	ClaimExpired(ctx context.Context, limit int) ([]ReservationExpiry, error)
}

// ReservationExpiry is the reaper's per-row result.
type ReservationExpiry struct {
	ReservationID uuid.UUID
	BuyerAuth0ID  string
	Amount        int64
}

// TransactionStore owns the append-only ledger.
type TransactionStore interface {
	// Insert writes a new ledger row. Returns ErrDuplicateIdempotencyKey
	// when (source_type, source_id, type) already exists — that's the
	// expected path for webhook replays and consumer redeliveries.
	Insert(ctx context.Context, tx *domain.Transaction) error

	// ListByBuyer returns (rows, total) for pagination on the buyer
	// history view. created_at DESC order.
	ListByBuyer(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Transaction, int, error)

	// GetByIdempotency looks up a row by idempotency key. Used to fetch
	// the existing row so we can report "already committed" on replay
	// instead of silently swallowing.
	GetByIdempotency(ctx context.Context, sourceType domain.SourceType, sourceID string, txType domain.TransactionType) (*domain.Transaction, error)
}
