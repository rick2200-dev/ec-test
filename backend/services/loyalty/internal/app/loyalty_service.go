// Package app is the loyalty service's application layer. It orchestrates
// domain operations across the ledger, account, and reservation stores
// inside transactional boundaries and emits events via Pub/Sub.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/domain"
	"github.com/Riku-KANO/ec-test/services/loyalty/internal/port"
)

// Service implements port.LoyaltyUseCase.
type Service struct {
	accounts     port.AccountStore
	transactions port.TransactionStore
	reservations port.ReservationStore
	tx           TxRunner
	publisher    pubsub.Publisher

	earnRateBps    int
	reservationTTL time.Duration
}

// TxRunner is a thin port over pkg/database.TxRunner so the app layer
// stays decoupled from pgx. It wraps a fn in a DB transaction, honoring
// the ambient context.
type TxRunner interface {
	RunTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// NewService wires the loyalty use cases. publisher may be nil (events
// drop silently), which is safe for tests or local dev without Pub/Sub.
// reservations may be nil only in tests that exercise the earn path in
// isolation — production main.go always wires a non-nil store.
func NewService(
	accounts port.AccountStore,
	transactions port.TransactionStore,
	reservations port.ReservationStore,
	tx TxRunner,
	publisher pubsub.Publisher,
	earnRateBps int,
	reservationTTL time.Duration,
) *Service {
	if reservationTTL <= 0 {
		reservationTTL = 30 * time.Minute
	}
	return &Service{
		accounts:       accounts,
		transactions:   transactions,
		reservations:   reservations,
		tx:             tx,
		publisher:      publisher,
		earnRateBps:    earnRateBps,
		reservationTTL: reservationTTL,
	}
}

// GetBalance returns the current account. Creates a zero-balance row if
// one does not exist yet so callers always see a well-formed response.
func (s *Service) GetBalance(ctx context.Context, buyerAuth0ID string) (*domain.Account, error) {
	if buyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}
	acct, err := s.accounts.Get(ctx, buyerAuth0ID)
	if err != nil {
		return nil, apperrors.Internal("failed to load account", err)
	}
	if acct == nil {
		return &domain.Account{BuyerAuth0ID: buyerAuth0ID}, nil
	}
	return acct, nil
}

// ListTransactions returns a paginated slice plus the total count.
func (s *Service) ListTransactions(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.Transaction, int, error) {
	if buyerAuth0ID == "" {
		return nil, 0, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	txs, total, err := s.transactions.ListByBuyer(ctx, buyerAuth0ID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list transactions", err)
	}
	return txs, total, nil
}

// AwardEarn credits points for a paid order. The amount is derived from
// paidSubtotal × earnRateBps / 10_000, floored to the nearest yen. The
// (order_paid, orderID, earn) idempotency key ensures webhook replays
// don't double-earn.
//
// Concurrency model: the account row is locked exclusively at the top
// of the tx (LockForUpdate). Two concurrent deliveries of the same
// order.paid — e.g. the synchronous order-svc call + the Pub/Sub
// subscriber — serialize on the row lock. The winner inserts the
// ledger row; the loser sees the row in its subsequent idempotency
// check and returns without touching the balance. This closes the
// over-credit window that the previous optimistic-lock scheme left
// open on the Insert-after-ApplyDelta error path.
//
// On a duplicate replay the existing row is returned (so the caller
// can treat the call as idempotent without caring about the
// difference between "first success" and "already done").
func (s *Service) AwardEarn(ctx context.Context, buyerAuth0ID, orderID string, paidSubtotal int64, currency string) (*domain.Transaction, error) {
	if buyerAuth0ID == "" || orderID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id and order_id are required")
	}
	amount := domain.ComputeEarnAmount(paidSubtotal, s.earnRateBps)
	if amount <= 0 {
		// Nothing to earn: zero-subtotal order or disabled rate.
		return nil, nil
	}

	var (
		created     *domain.Transaction
		wasNewRow   bool
	)
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		// 1. Serialize on the account row. Concurrent earn / reserve
		// calls for the same buyer block here until this tx commits.
		acct, err := s.accounts.LockForUpdate(ctx, buyerAuth0ID)
		if err != nil {
			return fmt.Errorf("lock account: %w", err)
		}

		// 2. Idempotency check. Because we hold the row lock, any
		// other writer for this buyer has either already committed
		// (row exists) or is still waiting behind our lock (can't
		// have inserted yet). Either way this check is authoritative.
		if existing, err := s.transactions.GetByIdempotency(ctx, domain.SourceTypeOrderPaid, orderID, domain.TransactionTypeEarn); err != nil {
			return fmt.Errorf("check idempotency: %w", err)
		} else if existing != nil {
			created = existing
			return nil
		}

		// 3. Apply balance + lifetime_earned. Still guarded by the
		// version check for defense in depth, but the row lock
		// already serializes writers so we won't see ErrOptimisticLock
		// in normal operation.
		newBalance := acct.Balance + amount
		if err := s.accounts.ApplyDelta(ctx, buyerAuth0ID, port.AccountDelta{
			BalanceDelta:        amount,
			LifetimeEarnedDelta: amount,
		}, acct.Version); err != nil {
			return fmt.Errorf("apply delta: %w", err)
		}

		// 4. Insert the ledger row. A duplicate here would only
		// happen if another process somehow bypassed the row lock
		// (e.g. a direct SQL patch); treating it as an invariant
		// violation keeps the ledger self-consistent by aborting
		// instead of silently swallowing.
		tx := &domain.Transaction{
			ID:           uuid.New(),
			BuyerAuth0ID: buyerAuth0ID,
			Type:         domain.TransactionTypeEarn,
			Amount:       amount,
			BalanceAfter: newBalance,
			SourceType:   domain.SourceTypeOrderPaid,
			SourceID:     orderID,
		}
		if err := s.transactions.Insert(ctx, tx); err != nil {
			if errors.Is(err, port.ErrDuplicateIdempotencyKey) {
				return fmt.Errorf("duplicate idempotency key while holding account lock: %w", err)
			}
			return fmt.Errorf("insert transaction: %w", err)
		}
		created = tx
		wasNewRow = true
		return nil
	})
	if err != nil {
		return nil, apperrors.Internal("failed to award earn", err)
	}

	// Only publish on a genuinely new award — duplicate replays stay
	// silent so downstream consumers don't see phantom PointsEarned
	// events.
	if wasNewRow && created != nil {
		s.publishEarned(ctx, created, orderID)
	}
	return created, nil
}

// ReservePoints holds `amount` in pending_redemption for the buyer.
// Does NOT write a ledger row — points are only burned on Commit. The
// account row lock both protects the spendable-balance check and
// serializes concurrent reservations from the same buyer.
func (s *Service) ReservePoints(ctx context.Context, buyerAuth0ID string, amount int64) (*domain.Reservation, error) {
	if buyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if s.reservations == nil {
		return nil, apperrors.Internal("reservation store not configured", nil)
	}

	var reservation *domain.Reservation
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		acct, err := s.accounts.LockForUpdate(ctx, buyerAuth0ID)
		if err != nil {
			return fmt.Errorf("lock account: %w", err)
		}
		if acct.Spendable() < amount {
			return domain.ErrInsufficientPoints
		}
		if err := s.accounts.ApplyDelta(ctx, buyerAuth0ID, port.AccountDelta{
			PendingRedemptionDelta: amount,
		}, acct.Version); err != nil {
			return fmt.Errorf("apply pending delta: %w", err)
		}
		r := &domain.Reservation{
			ID:           uuid.New(),
			BuyerAuth0ID: buyerAuth0ID,
			Amount:       amount,
			Status:       domain.ReservationStatusPending,
			ExpiresAt:    time.Now().Add(s.reservationTTL),
		}
		if err := s.reservations.Insert(ctx, r); err != nil {
			return fmt.Errorf("insert reservation: %w", err)
		}
		reservation = r
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientPoints) {
			return nil, err
		}
		return nil, apperrors.Internal("failed to reserve points", err)
	}
	return reservation, nil
}

// CommitPointsReservation burns the reserved points + accrues the
// earn on the same paid order. Both ledger writes have their own
// idempotency key; the reservation row transitions to committed
// inside the same tx so a concurrent Release / reaper can't unstick
// it after the ledger write. Returns AlreadyCommitted=true on replay.
func (s *Service) CommitPointsReservation(ctx context.Context, reservationID uuid.UUID, orderID, stripePaymentIntentID string, paidSubtotal int64, currency string) (*port.CommitPointsResult, error) {
	if reservationID == uuid.Nil {
		return nil, apperrors.BadRequest("reservation_id is required")
	}
	if orderID == "" {
		return nil, apperrors.BadRequest("order_id is required")
	}
	if s.reservations == nil {
		return nil, apperrors.Internal("reservation store not configured", nil)
	}

	var (
		out             port.CommitPointsResult
		publishedEarn   *domain.Transaction
		publishedRedeem *domain.Transaction
	)
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		reservation, err := s.reservations.GetByID(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("load reservation: %w", err)
		}
		if reservation == nil {
			return domain.ErrReservationNotFound
		}

		// Lock the account so the redeem + earn writes see a
		// consistent balance and concurrent AwardEarn / Reserve
		// calls serialize behind us.
		acct, err := s.accounts.LockForUpdate(ctx, reservation.BuyerAuth0ID)
		if err != nil {
			return fmt.Errorf("lock account: %w", err)
		}

		// Idempotent commit: if the redeem ledger row exists, the
		// commit has already happened — surface that and stop.
		existing, err := s.transactions.GetByIdempotency(ctx, domain.SourceTypeReservationCommit, reservationID.String(), domain.TransactionTypeRedeem)
		if err != nil {
			return fmt.Errorf("check redeem idempotency: %w", err)
		}
		if existing != nil {
			out.Redeemed = existing.Amount
			out.NewBalance = acct.Balance
			out.AlreadyCommitted = true
			// We still want to return the earned figure if the same
			// replay's earn row exists; callers persist it on the
			// order row for display.
			if earnRow, _ := s.transactions.GetByIdempotency(ctx, domain.SourceTypeOrderPaid, orderID, domain.TransactionTypeEarn); earnRow != nil {
				out.Earned = earnRow.Amount
			}
			return nil
		}

		if reservation.Status != domain.ReservationStatusPending {
			return domain.ErrReservationAlreadyTerminal
		}

		// 1. Flip reservation to committed.
		ok, err := s.reservations.UpdateStatus(ctx, reservationID, domain.ReservationStatusPending, domain.ReservationStatusCommitted)
		if err != nil {
			return fmt.Errorf("update reservation status: %w", err)
		}
		if !ok {
			return domain.ErrReservationAlreadyTerminal
		}

		// 2. Burn: debit balance + pending_redemption by amount,
		// bump lifetime_redeemed.
		burnBalanceAfter := acct.Balance - reservation.Amount
		if err := s.accounts.ApplyDelta(ctx, reservation.BuyerAuth0ID, port.AccountDelta{
			BalanceDelta:           -reservation.Amount,
			PendingRedemptionDelta: -reservation.Amount,
			LifetimeRedeemedDelta:  reservation.Amount,
		}, acct.Version); err != nil {
			return fmt.Errorf("apply redeem delta: %w", err)
		}
		redeemRow := &domain.Transaction{
			ID:            uuid.New(),
			BuyerAuth0ID:  reservation.BuyerAuth0ID,
			Type:          domain.TransactionTypeRedeem,
			Amount:        -reservation.Amount, // ledger sign: debit
			BalanceAfter:  burnBalanceAfter,
			SourceType:    domain.SourceTypeReservationCommit,
			SourceID:      reservationID.String(),
			ReservationID: &reservationID,
		}
		if err := s.transactions.Insert(ctx, redeemRow); err != nil {
			return fmt.Errorf("insert redeem row: %w", err)
		}

		// 3. Accrue: if paid_subtotal × rate produces a positive
		// earn, credit balance + lifetime_earned. The account
		// snapshot was captured before the burn, so we compute the
		// new balance from the intermediate burnBalanceAfter.
		earnAmount := domain.ComputeEarnAmount(paidSubtotal, s.earnRateBps)
		earnBalanceAfter := burnBalanceAfter
		if earnAmount > 0 {
			// Idempotency check on earn so a webhook replay that
			// already inserted the earn row (but maybe lost on the
			// redeem row) short-circuits cleanly.
			if existingEarn, err := s.transactions.GetByIdempotency(ctx, domain.SourceTypeOrderPaid, orderID, domain.TransactionTypeEarn); err != nil {
				return fmt.Errorf("check earn idempotency: %w", err)
			} else if existingEarn == nil {
				earnBalanceAfter = burnBalanceAfter + earnAmount
				if err := s.accounts.ApplyDelta(ctx, reservation.BuyerAuth0ID, port.AccountDelta{
					BalanceDelta:        earnAmount,
					LifetimeEarnedDelta: earnAmount,
				}, acct.Version+1); err != nil {
					return fmt.Errorf("apply earn delta: %w", err)
				}
				earnRow := &domain.Transaction{
					ID:           uuid.New(),
					BuyerAuth0ID: reservation.BuyerAuth0ID,
					Type:         domain.TransactionTypeEarn,
					Amount:       earnAmount,
					BalanceAfter: earnBalanceAfter,
					SourceType:   domain.SourceTypeOrderPaid,
					SourceID:     orderID,
				}
				if err := s.transactions.Insert(ctx, earnRow); err != nil {
					return fmt.Errorf("insert earn row: %w", err)
				}
				publishedEarn = earnRow
			} else {
				// Earn already applied elsewhere (sync call beat
				// us). Just report the amount; don't double-credit.
				out.Earned = existingEarn.Amount
			}
		}

		out.Redeemed = reservation.Amount
		if publishedEarn != nil {
			out.Earned = publishedEarn.Amount
		}
		out.NewBalance = earnBalanceAfter
		publishedRedeem = redeemRow
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) || errors.Is(err, domain.ErrReservationAlreadyTerminal) {
			return nil, err
		}
		return nil, apperrors.Internal("failed to commit reservation", err)
	}
	if publishedRedeem != nil {
		s.publishRedeemed(ctx, publishedRedeem, orderID)
	}
	if publishedEarn != nil {
		s.publishEarned(ctx, publishedEarn, orderID)
	}
	return &out, nil
}

// ReleasePointsReservation returns pending points to spendable
// balance. No ledger row is written — Reserve never burned anything,
// so Release is purely a counter adjustment.
func (s *Service) ReleasePointsReservation(ctx context.Context, reservationID uuid.UUID, reason string) (bool, error) {
	if reservationID == uuid.Nil {
		return false, apperrors.BadRequest("reservation_id is required")
	}
	if s.reservations == nil {
		return false, apperrors.Internal("reservation store not configured", nil)
	}
	var released bool
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		reservation, err := s.reservations.GetByID(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("load reservation: %w", err)
		}
		if reservation == nil {
			return domain.ErrReservationNotFound
		}
		if reservation.Status != domain.ReservationStatusPending {
			// Already terminal — no-op, not an error.
			return nil
		}
		acct, err := s.accounts.LockForUpdate(ctx, reservation.BuyerAuth0ID)
		if err != nil {
			return fmt.Errorf("lock account: %w", err)
		}
		ok, err := s.reservations.UpdateStatus(ctx, reservationID, domain.ReservationStatusPending, domain.ReservationStatusReleased)
		if err != nil {
			return fmt.Errorf("update status: %w", err)
		}
		if !ok {
			return nil // lost the race, treat as released elsewhere
		}
		if err := s.accounts.ApplyDelta(ctx, reservation.BuyerAuth0ID, port.AccountDelta{
			PendingRedemptionDelta: -reservation.Amount,
		}, acct.Version); err != nil {
			return fmt.Errorf("apply release delta: %w", err)
		}
		released = true
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return false, err
		}
		return false, apperrors.Internal("failed to release reservation", err)
	}
	slog.Info("loyalty reservation released", "reservation_id", reservationID, "reason", reason, "released", released)
	return released, nil
}

// ApplyCancellation reverses the effects of a paid order on
// cancellation. For each non-zero amount in the input we append one
// compensation ledger row:
//   - refund (positive amount): credits back the points the buyer
//     redeemed on this order.
//   - reverse_earn (negative amount): removes the earn credited on
//     order.paid. Capped at the buyer's current balance to respect
//     balance >= 0; any leftover is logged for ops review.
//
// Idempotent via (order_cancelled, orderID, refund) and
// (order_cancelled, orderID, reverse_earn) UNIQUE keys: a redelivered
// order.cancelled event writes nothing the second time.
func (s *Service) ApplyCancellation(ctx context.Context, in port.CancelPointsInput) (*port.CancelPointsResult, error) {
	if in.BuyerAuth0ID == "" || in.OrderID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id and order_id are required")
	}
	if in.PointDiscountAmount < 0 || in.PointsEarned < 0 {
		return nil, apperrors.BadRequest("cancellation amounts must be non-negative")
	}
	if in.PointDiscountAmount == 0 && in.PointsEarned == 0 {
		return &port.CancelPointsResult{}, nil
	}

	var out port.CancelPointsResult
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		acct, err := s.accounts.LockForUpdate(ctx, in.BuyerAuth0ID)
		if err != nil {
			return fmt.Errorf("lock account: %w", err)
		}
		version := acct.Version
		running := acct.Balance

		if in.PointDiscountAmount > 0 {
			existing, err := s.transactions.GetByIdempotency(ctx, domain.SourceTypeOrderCancelled, in.OrderID, domain.TransactionTypeRefund)
			if err != nil {
				return fmt.Errorf("check refund idempotency: %w", err)
			}
			if existing == nil {
				running += in.PointDiscountAmount
				if err := s.accounts.ApplyDelta(ctx, in.BuyerAuth0ID, port.AccountDelta{
					BalanceDelta: in.PointDiscountAmount,
					// Lifetime counters intentionally untouched: ops
					// preferred "redeemed" to capture buyer intent at
					// purchase time, not final settlement.
				}, version); err != nil {
					return fmt.Errorf("apply refund delta: %w", err)
				}
				version++
				row := &domain.Transaction{
					ID:           uuid.New(),
					BuyerAuth0ID: in.BuyerAuth0ID,
					Type:         domain.TransactionTypeRefund,
					Amount:       in.PointDiscountAmount,
					BalanceAfter: running,
					SourceType:   domain.SourceTypeOrderCancelled,
					SourceID:     in.OrderID,
					Note:         "refund of redeemed points on order cancellation",
				}
				if err := s.transactions.Insert(ctx, row); err != nil {
					return fmt.Errorf("insert refund row: %w", err)
				}
				out.Refunded = in.PointDiscountAmount
			}
		}

		if in.PointsEarned > 0 {
			existing, err := s.transactions.GetByIdempotency(ctx, domain.SourceTypeOrderCancelled, in.OrderID, domain.TransactionTypeReverseEarn)
			if err != nil {
				return fmt.Errorf("check reverse_earn idempotency: %w", err)
			}
			if existing == nil {
				reversal := in.PointsEarned
				if reversal > running {
					slog.Warn("reverse_earn capped by available balance",
						"buyer", in.BuyerAuth0ID, "order_id", in.OrderID,
						"wanted", in.PointsEarned, "applied", running)
					reversal = running
				}
				if reversal > 0 {
					running -= reversal
					if err := s.accounts.ApplyDelta(ctx, in.BuyerAuth0ID, port.AccountDelta{
						BalanceDelta:        -reversal,
						LifetimeEarnedDelta: -reversal,
					}, version); err != nil {
						return fmt.Errorf("apply reverse_earn delta: %w", err)
					}
					version++
					row := &domain.Transaction{
						ID:           uuid.New(),
						BuyerAuth0ID: in.BuyerAuth0ID,
						Type:         domain.TransactionTypeReverseEarn,
						Amount:       -reversal,
						BalanceAfter: running,
						SourceType:   domain.SourceTypeOrderCancelled,
						SourceID:     in.OrderID,
						Note:         "reversal of earn on order cancellation",
					}
					if err := s.transactions.Insert(ctx, row); err != nil {
						return fmt.Errorf("insert reverse_earn row: %w", err)
					}
					out.Reversed = reversal
				}
			}
		}
		out.NewBalance = running
		return nil
	})
	if err != nil {
		return nil, apperrors.Internal("failed to apply cancellation", err)
	}
	if out.Refunded > 0 || out.Reversed > 0 {
		slog.Info("loyalty cancellation applied",
			"buyer", in.BuyerAuth0ID, "order_id", in.OrderID,
			"refunded", out.Refunded, "reversed", out.Reversed)
	}
	return &out, nil
}

// AdjustPoints applies a signed admin-initiated delta to a buyer's
// balance and appends an `adjust` ledger row. Positive credits the
// account; negative debits it (capped by spendable balance). reason and
// admin_user_id are persisted on the row for audit. Each call gets a
// fresh source_id (ledger uuid) since admin intent does not have the
// replay semantics that webhook/pubsub paths do — every click is an
// explicit action.
func (s *Service) AdjustPoints(ctx context.Context, in port.AdjustPointsInput) (*port.AdjustPointsResult, error) {
	if in.BuyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if in.Amount == 0 {
		return nil, domain.ErrInvalidAmount
	}
	if in.Reason == "" {
		return nil, apperrors.BadRequest("reason is required")
	}

	var result port.AdjustPointsResult
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		if err := s.accounts.EnsureExists(ctx, in.BuyerAuth0ID); err != nil {
			return fmt.Errorf("ensure account: %w", err)
		}
		acct, err := s.accounts.LockForUpdate(ctx, in.BuyerAuth0ID)
		if err != nil {
			return fmt.Errorf("lock account: %w", err)
		}
		if in.Amount < 0 && acct.Spendable()+in.Amount < 0 {
			return domain.ErrInsufficientPoints
		}

		newBalance := acct.Balance + in.Amount
		if err := s.accounts.ApplyDelta(ctx, in.BuyerAuth0ID, port.AccountDelta{
			BalanceDelta: in.Amount,
		}, acct.Version); err != nil {
			return fmt.Errorf("apply adjust delta: %w", err)
		}

		row := &domain.Transaction{
			ID:           uuid.New(),
			BuyerAuth0ID: in.BuyerAuth0ID,
			Type:         domain.TransactionTypeAdjust,
			Amount:       in.Amount,
			BalanceAfter: newBalance,
			SourceType:   domain.SourceTypeAdminAdjust,
			Note:         formatAdjustNote(in.AdminUserID, in.Reason),
		}
		// source_id is the row's own uuid: admin actions are not idempotent
		// across retries, so each submission intentionally writes a new
		// ledger row. Using the tx ID keeps the (source_type, source_id,
		// type) key unique without requiring an external idempotency token.
		row.SourceID = row.ID.String()

		if err := s.transactions.Insert(ctx, row); err != nil {
			return fmt.Errorf("insert adjust row: %w", err)
		}
		result.Transaction = row
		result.NewBalance = newBalance
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientPoints) {
			return nil, err
		}
		return nil, apperrors.Internal("failed to adjust points", err)
	}
	slog.Info("loyalty points adjusted",
		"buyer", in.BuyerAuth0ID, "amount", in.Amount,
		"admin", in.AdminUserID, "new_balance", result.NewBalance)
	return &result, nil
}

// formatAdjustNote packs admin_user_id + reason into the ledger note
// field. The Transaction schema does not have a dedicated admin column,
// and adding one for a single code path wasn't worth a migration — the
// prefix is machine-parseable if ops ever needs to filter by admin.
func formatAdjustNote(adminUserID, reason string) string {
	if adminUserID == "" {
		return reason
	}
	return "admin=" + adminUserID + ": " + reason
}

// ExpireStaleReservations is driven by the reaper goroutine. Pending
// reservations past their TTL are flipped to expired and each
// buyer's pending_redemption decremented.
func (s *Service) ExpireStaleReservations(ctx context.Context) (int, error) {
	if s.reservations == nil {
		return 0, nil
	}
	const batchSize = 200
	var affected int
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		rows, err := s.reservations.ClaimExpired(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("claim expired: %w", err)
		}
		for _, r := range rows {
			acct, err := s.accounts.LockForUpdate(ctx, r.BuyerAuth0ID)
			if err != nil {
				return fmt.Errorf("lock account %s: %w", r.BuyerAuth0ID, err)
			}
			if err := s.accounts.ApplyDelta(ctx, r.BuyerAuth0ID, port.AccountDelta{
				PendingRedemptionDelta: -r.Amount,
			}, acct.Version); err != nil {
				return fmt.Errorf("release pending for %s: %w", r.BuyerAuth0ID, err)
			}
		}
		affected = len(rows)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if affected > 0 {
		slog.Info("loyalty reservations expired", "count", affected)
	}
	return affected, nil
}

// publishRedeemed emits PointsRedeemed on a successful Commit.
func (s *Service) publishRedeemed(ctx context.Context, tx *domain.Transaction, orderID string) {
	if s.publisher == nil {
		return
	}
	pubsub.PublishEvent(ctx, s.publisher, domain.EventTypePointsRedeemed, domain.TopicLoyaltyEvents, map[string]any{
		"transaction_id": tx.ID.String(),
		"buyer_auth0_id": tx.BuyerAuth0ID,
		// amount on the wire is the positive spend, not the signed
		// ledger value, so consumers don't need to know the sign
		// convention.
		"amount":        -tx.Amount,
		"balance_after": tx.BalanceAfter,
		"order_id":      orderID,
		"redeemed_at":   tx.CreatedAt,
	})
}

// publishEarned emits PointsEarned. Payload is JSON for the MVP; the
// proto-backed version lands when the coupon/loyalty proto gen runs.
func (s *Service) publishEarned(ctx context.Context, tx *domain.Transaction, orderID string) {
	if s.publisher == nil {
		return
	}
	pubsub.PublishEvent(ctx, s.publisher, domain.EventTypePointsEarned, domain.TopicLoyaltyEvents, map[string]any{
		"transaction_id": tx.ID.String(),
		"buyer_auth0_id": tx.BuyerAuth0ID,
		"amount":         tx.Amount,
		"balance_after":  tx.BalanceAfter,
		"order_id":       orderID,
		"earned_at":      tx.CreatedAt,
	})
	slog.Debug("points earned event published", "buyer", tx.BuyerAuth0ID, "order_id", orderID, "amount", tx.Amount)
}
