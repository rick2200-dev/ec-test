// Package app is the coupon service's application layer. The service
// orchestrates the three stores (coupons, reservations, redemptions)
// inside transactional boundaries and publishes coupon-events.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/domain"
	"github.com/Riku-KANO/ec-test/services/coupon/internal/port"
)

// TxRunner mirrors pkg/database.TxRunner so app stays decoupled from pgx.
type TxRunner interface {
	RunTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Clock is the app's monotonic clock seam. Tests override it to
// assert TTL edges without waiting on the wall clock.
type Clock interface {
	Now() time.Time
}

// RealClock uses time.Now(). Default when no clock is injected.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// Service implements port.CouponUseCase.
type Service struct {
	coupons      port.CouponStore
	reservations port.ReservationStore
	redemptions  port.RedemptionStore
	tx           TxRunner
	publisher    pubsub.Publisher
	clock        Clock

	reservationTTL time.Duration
}

// NewService wires the coupon use cases. publisher may be nil (events
// drop silently — safe for tests and local dev).
func NewService(
	coupons port.CouponStore,
	reservations port.ReservationStore,
	redemptions port.RedemptionStore,
	tx TxRunner,
	publisher pubsub.Publisher,
	reservationTTL time.Duration,
) *Service {
	return &Service{
		coupons:        coupons,
		reservations:   reservations,
		redemptions:    redemptions,
		tx:             tx,
		publisher:      publisher,
		clock:          RealClock{},
		reservationTTL: reservationTTL,
	}
}

// SetClock is a test hook. Production code uses the RealClock that
// NewService installs by default.
func (s *Service) SetClock(c Clock) { s.clock = c }

// --- Admin ---

func (s *Service) CreateCoupon(ctx context.Context, in port.CreateCouponInput) (*domain.Coupon, error) {
	if err := validateCreateInput(in); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	coupon := &domain.Coupon{
		ID:                 uuid.New(),
		Code:               strings.TrimSpace(in.Code),
		IssuerType:         in.IssuerType,
		IssuerID:           in.IssuerID,
		DiscountType:       in.DiscountType,
		DiscountPercentBps: in.DiscountPercentBps,
		DiscountAmount:     in.DiscountAmount,
		Currency:           defaultStr(in.Currency, "JPY"),
		MinOrderAmount:     in.MinOrderAmount,
		MaxDiscountAmount:  in.MaxDiscountAmount,
		ValidFrom:          now,
		UsageLimitTotal:    in.UsageLimitTotal,
		UsageLimitPerUser:  in.UsageLimitPerUser,
		Status:             domain.CouponStatusActive,
		Title:              in.Title,
		Description:        in.Description,
	}
	if in.ValidFromSet {
		coupon.ValidFrom = time.Unix(in.ValidFromUnix, 0).UTC()
	}
	if in.ExpiresAtUnix != nil {
		t := time.Unix(*in.ExpiresAtUnix, 0).UTC()
		coupon.ExpiresAt = &t
	}

	if err := s.coupons.Insert(ctx, coupon); err != nil {
		return nil, apperrors.Internal("failed to insert coupon", err)
	}

	// Best-effort publish. Failures are logged by pubsub internals.
	pubsub.PublishEvent(ctx, s.publisher, domain.EventTypeCouponIssued, domain.TopicCouponEvents, map[string]any{
		"coupon_id":   coupon.ID.String(),
		"code":        coupon.Code,
		"issuer_type": string(coupon.IssuerType),
		"issued_at":   now,
	})
	slog.Info("coupon created", "id", coupon.ID, "code", coupon.Code, "issuer_type", coupon.IssuerType)
	return coupon, nil
}

func (s *Service) ListCoupons(ctx context.Context, f port.ListCouponsFilter) ([]domain.Coupon, int, error) {
	limit, offset := clampPage(f.Limit, f.Offset)
	coupons, total, err := s.coupons.List(ctx, f.Status, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list coupons", err)
	}
	return coupons, total, nil
}

func (s *Service) GetCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	c, err := s.coupons.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to load coupon", err)
	}
	if c == nil {
		return nil, domain.ErrCouponNotFound
	}
	return c, nil
}

func (s *Service) RevokeCoupon(ctx context.Context, id uuid.UUID) (*domain.Coupon, error) {
	c, err := s.coupons.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to load coupon", err)
	}
	if c == nil {
		return nil, domain.ErrCouponNotFound
	}
	if c.Status == domain.CouponStatusRevoked {
		return c, nil
	}
	if err := s.coupons.SetStatus(ctx, id, domain.CouponStatusRevoked); err != nil {
		return nil, apperrors.Internal("failed to revoke coupon", err)
	}
	c.Status = domain.CouponStatusRevoked
	pubsub.PublishEvent(ctx, s.publisher, domain.EventTypeCouponRevoked, domain.TopicCouponEvents, map[string]any{
		"coupon_id":  c.ID.String(),
		"code":       c.Code,
		"revoked_at": s.clock.Now(),
	})
	slog.Info("coupon revoked", "id", c.ID, "code", c.Code)
	return c, nil
}

func (s *Service) GetCouponStats(ctx context.Context, id uuid.UUID) (*port.CouponStats, error) {
	c, err := s.coupons.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to load coupon", err)
	}
	if c == nil {
		return nil, domain.ErrCouponNotFound
	}
	count, total, err := s.redemptions.StatsByCoupon(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to load stats", err)
	}
	pending, err := s.reservations.CountPendingByCoupon(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to count pending reservations", err)
	}
	return &port.CouponStats{
		CouponID:                id,
		RedeemedCount:           count,
		TotalDiscountApplied:    total,
		PendingReservationCount: pending,
	}, nil
}

// --- Buyer ---

// PreviewCoupon runs the full validity checks and computes a discount
// amount, but writes nothing. Used by the buyer UI for the "apply
// code" feedback loop before the cart actually commits.
func (s *Service) PreviewCoupon(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []domain.SellerSubtotal, currency string) (*port.PreviewResult, error) {
	c, applicableSeller, applicableSubtotal, err := s.resolveCoupon(ctx, code, buyerAuth0ID, sellerSubtotals, currency, false)
	if err != nil {
		return nil, err
	}
	discount := c.ComputeDiscount(applicableSubtotal)
	result := &port.PreviewResult{
		CouponID:           c.ID,
		Title:              c.Title,
		Description:        c.Description,
		DiscountAmount:     discount,
		ApplicableSellerID: applicableSeller,
	}
	if c.ExpiresAt != nil {
		s := c.ExpiresAt.UTC().Format(time.RFC3339)
		result.ExpiresAt = &s
	}
	return result, nil
}

func (s *Service) ListMyRedemptions(ctx context.Context, buyerAuth0ID string, limit, offset int) ([]domain.CouponRedemption, int, error) {
	if buyerAuth0ID == "" {
		return nil, 0, apperrors.BadRequest("buyer_auth0_id is required")
	}
	limit, offset = clampPage(limit, offset)
	rows, total, err := s.redemptions.ListByBuyer(ctx, buyerAuth0ID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list redemptions", err)
	}
	return rows, total, nil
}

// --- Internal (order-svc) ---

// ReserveCoupon runs inside a single Tx:
//  1. lock coupon row (GetByCodeForUpdate),
//  2. validate everything (status, validity, currency, min_order, per-user usage),
//  3. bump coupons.usage_count if below the total limit,
//  4. insert coupon_reservations with TTL expires_at.
//
// Holding the coupon row lock for the whole Tx gives us pessimistic
// concurrency against two buyers racing for the last use of the same
// code. The per-user check uses CountByBuyerAndCoupon on the
// redemptions ledger — reservations aren't counted, so a buyer who
// abandons checkout and retries immediately still gets through.
func (s *Service) ReserveCoupon(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []domain.SellerSubtotal, currency string) (*domain.CouponReservation, error) {
	var reservation *domain.CouponReservation
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		c, applicableSeller, applicableSubtotal, err := s.resolveCoupon(ctx, code, buyerAuth0ID, sellerSubtotals, currency, true)
		if err != nil {
			return err
		}

		discount := c.ComputeDiscount(applicableSubtotal)
		if discount <= 0 {
			return domain.ErrInvalidCouponInput
		}

		if err := s.coupons.IncrementUsageIfBelowLimit(ctx, c.ID); err != nil {
			if errors.Is(err, port.ErrUsageLimitExceeded) {
				return domain.ErrCouponUsageLimitHit
			}
			return fmt.Errorf("increment usage: %w", err)
		}

		r := &domain.CouponReservation{
			ID:             uuid.New(),
			CouponID:       c.ID,
			BuyerAuth0ID:   buyerAuth0ID,
			DiscountAmount: discount,
			Currency:       c.Currency,
			Status:         domain.ReservationStatusPending,
			ExpiresAt:      s.clock.Now().Add(s.reservationTTL),
		}
		if err := s.reservations.Insert(ctx, r); err != nil {
			return fmt.Errorf("insert reservation: %w", err)
		}
		reservation = r
		_ = applicableSeller // applicable_seller is returned upstream via the response struct, not the reservation row (platform-only in MVP, so always empty).
		return nil
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return reservation, nil
}

// CommitReservation flips the reservation to committed and appends a
// redemption ledger row. On idempotent replay (webhook delivered
// twice) the redemption insert loses to the first writer's UNIQUE
// constraint — in that case we return the existing row and report
// already_committed=true.
func (s *Service) CommitReservation(ctx context.Context, reservationID, orderID uuid.UUID, stripePaymentIntentID string) (*port.CommitResult, error) {
	var result *port.CommitResult
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		reservation, err := s.reservations.GetByID(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("load reservation: %w", err)
		}
		if reservation == nil {
			return domain.ErrReservationNotFound
		}

		// Idempotent commit: if a redemption already exists for this
		// (coupon_id, order_id), short-circuit.
		existing, err := s.redemptions.GetByCouponAndOrder(ctx, reservation.CouponID, orderID)
		if err != nil {
			return fmt.Errorf("check existing redemption: %w", err)
		}
		if existing != nil {
			result = &port.CommitResult{RedemptionID: existing.ID, AlreadyCommitted: true}
			return nil
		}

		if reservation.Status != domain.ReservationStatusPending {
			return domain.ErrReservationAlreadyTerminal
		}

		ok, err := s.reservations.UpdateStatus(ctx, reservationID, domain.ReservationStatusPending, domain.ReservationStatusCommitted)
		if err != nil {
			return fmt.Errorf("update reservation status: %w", err)
		}
		if !ok {
			// Lost the race against a concurrent release/expire.
			return domain.ErrReservationAlreadyTerminal
		}

		redemption := &domain.CouponRedemption{
			ID:              uuid.New(),
			CouponID:        reservation.CouponID,
			BuyerAuth0ID:    reservation.BuyerAuth0ID,
			OrderID:         orderID,
			DiscountApplied: reservation.DiscountAmount,
			ReservationID:   reservationID,
		}
		if err := s.redemptions.Insert(ctx, redemption); err != nil {
			if errors.Is(err, port.ErrDuplicateRedemption) {
				// Another commit slipped in between our check and our
				// insert — return that one.
				existing, lookupErr := s.redemptions.GetByCouponAndOrder(ctx, reservation.CouponID, orderID)
				if lookupErr != nil {
					return fmt.Errorf("lookup race-winner redemption: %w", lookupErr)
				}
				if existing == nil {
					return fmt.Errorf("race-winner redemption vanished")
				}
				result = &port.CommitResult{RedemptionID: existing.ID, AlreadyCommitted: true}
				return nil
			}
			return fmt.Errorf("insert redemption: %w", err)
		}
		result = &port.CommitResult{RedemptionID: redemption.ID}
		return nil
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	if result != nil && !result.AlreadyCommitted {
		pubsub.PublishEvent(ctx, s.publisher, domain.EventTypeCouponRedeemed, domain.TopicCouponEvents, map[string]any{
			"redemption_id": result.RedemptionID.String(),
			"order_id":      orderID.String(),
			"stripe_pi_id":  stripePaymentIntentID,
			"committed_at":  s.clock.Now(),
		})
	}
	return result, nil
}

// ReleaseReservation is called on checkout failure / cancellation. It
// flips the reservation to released and decrements usage_count so the
// seat is available for someone else. Idempotent: a non-pending row
// returns (false, nil).
func (s *Service) ReleaseReservation(ctx context.Context, reservationID uuid.UUID, reason string) (bool, error) {
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
			// Already terminal — no-op but not an error.
			return nil
		}
		ok, err := s.reservations.UpdateStatus(ctx, reservationID, domain.ReservationStatusPending, domain.ReservationStatusReleased)
		if err != nil {
			return fmt.Errorf("update status: %w", err)
		}
		if !ok {
			return nil // lost the race; treat as released elsewhere
		}
		if err := s.coupons.DecrementUsage(ctx, reservation.CouponID); err != nil {
			return fmt.Errorf("decrement usage: %w", err)
		}
		released = true
		return nil
	})
	if err != nil {
		return false, mapDomainError(err)
	}
	slog.Info("coupon reservation released", "reservation_id", reservationID, "reason", reason, "released", released)
	return released, nil
}

// RefundRedemption credits `share` of the coupon discount back when
// one order in a (potentially multi-order) cart is cancelled. The
// redemption is cart-wide — commit wrote a single row keyed on the
// anchor order_id, and the per-order discount distribution is
// tracked by accumulating refunded_amount here. The coupon seat
// (usage_count) is only released on the transition that brings
// refunded_amount equal to discount_applied, which happens on the
// final cancel of a partially-cancelled cart or on the first cancel
// of a single-order cart.
//
// Idempotency:
//   - reservation_id resolves to exactly one redemption row; a
//     cancel event for a non-coupon order passes reservationID =
//     uuid.Nil and we short-circuit with Skipped=true.
//   - `order.cancelled` is fired per-order; each delivery carries its
//     own share. The ApplyPartialRefund SQL is a single atomic
//     UPDATE guarded by refunded_at IS NULL, so a redelivered event
//     for the same order can't double-count.
//   - Clamping at discount_applied belt-and-suspenders a malformed
//     caller that sends a share larger than what remains.
func (s *Service) RefundRedemption(ctx context.Context, reservationID, orderID uuid.UUID, share int64, reason string) (*port.RefundResult, error) {
	if reservationID == uuid.Nil {
		// Non-coupon order cancellation — nothing to do.
		return &port.RefundResult{Skipped: true}, nil
	}
	if orderID == uuid.Nil {
		return nil, apperrors.BadRequest("order_id is required")
	}
	if share < 0 {
		return nil, apperrors.BadRequest("share must be non-negative")
	}

	var out port.RefundResult
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		redemption, err := s.redemptions.GetByReservationID(ctx, reservationID)
		if err != nil {
			return fmt.Errorf("load redemption by reservation: %w", err)
		}
		if redemption == nil {
			// Reservation exists on the order side but no redemption
			// was committed (e.g. payment failed before Commit).
			// Nothing to reverse.
			out.Skipped = true
			return nil
		}
		out.RedemptionID = redemption.ID

		if share == 0 {
			// Zero-share cancel on a coupon order is a no-op, but we
			// still report the redemption so callers can log sensibly.
			return nil
		}

		// (1) Claim the (redemption, cancelled_order) pair via a
		// UNIQUE-constrained ledger insert. The first INSERT for a
		// given cancelled order wins; a redelivered
		// order.cancelled event hits ErrDuplicateRefundEvent and
		// short-circuits as AlreadyRefunded *for this order*. This
		// is what makes the per-order refund idempotent — without
		// it, a redelivered event would pass the refunded_at IS
		// NULL guard on ApplyPartialRefund and add the same share
		// a second time.
		if err := s.redemptions.InsertRefundEvent(ctx, redemption.ID, orderID, share, reason); err != nil {
			if errors.Is(err, port.ErrDuplicateRefundEvent) {
				out.AlreadyRefunded = true
				return nil
			}
			return fmt.Errorf("insert refund event: %w", err)
		}

		// (2) Now that this order's share is claimed exactly once,
		// apply it to the redemption's accumulated refunded_amount.
		// The repo still caps at discount_applied defensively.
		applied, fully, alreadyFull, err := s.redemptions.ApplyPartialRefund(ctx, redemption.ID, share, reason)
		if err != nil {
			return fmt.Errorf("apply partial refund: %w", err)
		}
		if alreadyFull {
			// Another order already brought the redemption to full
			// refund in a concurrent tx — our event row stays on the
			// ledger as audit, but we didn't actually move the
			// accumulator.
			out.AlreadyRefunded = true
			return nil
		}
		out.AppliedAmount = applied
		out.FullyRefunded = fully
		if fully {
			// Releasing the seat happens exactly on the transition
			// to fully-refunded. DecrementUsage is floored at zero,
			// so a would-be double-release from a racing tx is
			// harmless.
			if err := s.coupons.DecrementUsage(ctx, redemption.CouponID); err != nil {
				return fmt.Errorf("decrement usage on full refund: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, apperrors.Internal("failed to refund redemption", err)
	}
	if out.RedemptionID != uuid.Nil && out.AppliedAmount > 0 {
		slog.Info("coupon redemption refund applied",
			"redemption_id", out.RedemptionID,
			"order_id", orderID,
			"applied", out.AppliedAmount,
			"fully_refunded", out.FullyRefunded,
			"reason", reason,
		)
	}
	return &out, nil
}

// ExpireStaleReservations is driven by the reaper goroutine. Pending
// reservations past their TTL are flipped to expired and their
// coupons.usage_count is decremented. Batched to avoid long tx holds.
func (s *Service) ExpireStaleReservations(ctx context.Context) (int, error) {
	const batchSize = 200
	var affected int
	err := s.tx.RunTx(ctx, func(ctx context.Context) error {
		rows, err := s.reservations.ClaimExpired(ctx, batchSize)
		if err != nil {
			return fmt.Errorf("claim expired: %w", err)
		}
		for _, r := range rows {
			if err := s.coupons.DecrementUsage(ctx, r.CouponID); err != nil {
				return fmt.Errorf("decrement usage for expired reservation %s: %w", r.ReservationID, err)
			}
		}
		affected = len(rows)
		return nil
	})
	if err != nil {
		return 0, err
	}
	if affected > 0 {
		slog.Info("coupon reservations expired", "count", affected)
	}
	return affected, nil
}

// --- helpers ---

// resolveCoupon loads the coupon (optionally locking the row for
// Reserve) and runs the full validity gate: status, validity window,
// currency, min_order, per-user usage limit. Returns the applicable
// seller (nil for platform coupons) and the subtotal that applies to
// that scope.
func (s *Service) resolveCoupon(ctx context.Context, code, buyerAuth0ID string, sellerSubtotals []domain.SellerSubtotal, currency string, lock bool) (*domain.Coupon, *uuid.UUID, int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil, 0, apperrors.BadRequest("code is required")
	}
	if buyerAuth0ID == "" {
		return nil, nil, 0, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if currency == "" {
		currency = "JPY"
	}

	var c *domain.Coupon
	var err error
	if lock {
		// MVP is platform-only so issuer_id is NULL; future seller
		// coupons would look up with the seller_id hint.
		c, err = s.coupons.GetByCodeForUpdate(ctx, domain.IssuerTypePlatform, nil, code)
	} else {
		// Non-locking lookup: use List with a code filter? Simpler:
		// reuse GetByCodeForUpdate — on preview there's no meaningful
		// contention and the lock is released with the tx.
		c, err = s.coupons.GetByCodeForUpdate(ctx, domain.IssuerTypePlatform, nil, code)
	}
	if err != nil {
		return nil, nil, 0, apperrors.Internal("failed to load coupon", err)
	}
	if c == nil {
		return nil, nil, 0, domain.ErrCouponNotFound
	}
	if c.Status == domain.CouponStatusRevoked {
		return nil, nil, 0, domain.ErrCouponRevoked
	}
	now := s.clock.Now()
	if !c.IsWithinValidity(now) {
		if c.ExpiresAt != nil && now.After(*c.ExpiresAt) {
			return nil, nil, 0, domain.ErrCouponExpired
		}
		return nil, nil, 0, domain.ErrCouponNotStarted
	}
	if !strings.EqualFold(c.Currency, currency) {
		return nil, nil, 0, domain.ErrCurrencyMismatch
	}

	// MVP: platform coupons apply to cart-wide subtotal.
	var applicableSubtotal int64
	for _, s := range sellerSubtotals {
		applicableSubtotal += s.Subtotal
	}
	if applicableSubtotal < c.MinOrderAmount {
		return nil, nil, 0, domain.ErrMinOrderNotMet
	}

	// Per-user limit (checked against redemptions only, not reservations).
	if c.UsageLimitPerUser != nil {
		count, err := s.redemptions.CountByBuyerAndCoupon(ctx, buyerAuth0ID, c.ID)
		if err != nil {
			return nil, nil, 0, apperrors.Internal("failed to check per-user limit", err)
		}
		if count >= *c.UsageLimitPerUser {
			return nil, nil, 0, domain.ErrCouponPerUserLimit
		}
	}

	return c, nil, applicableSubtotal, nil
}

// mapDomainError preserves *AppError pass-through while tagging domain
// sentinels that need specific HTTP mappings. Handlers' error_mapper
// picks up the sentinels and returns the right status/code.
func mapDomainError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return err
	}
	// Let domain sentinels flow through unchanged — the handler layer
	// translates them to HTTP. Anything else is infra noise → 500.
	for _, sentinel := range []error{
		domain.ErrCouponNotFound,
		domain.ErrCouponExpired,
		domain.ErrCouponRevoked,
		domain.ErrCouponNotStarted,
		domain.ErrCouponUsageLimitHit,
		domain.ErrCouponPerUserLimit,
		domain.ErrMinOrderNotMet,
		domain.ErrCurrencyMismatch,
		domain.ErrInvalidCouponInput,
		domain.ErrReservationNotFound,
		domain.ErrReservationAlreadyTerminal,
	} {
		if errors.Is(err, sentinel) {
			return err
		}
	}
	return apperrors.Internal("coupon operation failed", err)
}

func validateCreateInput(in port.CreateCouponInput) error {
	if strings.TrimSpace(in.Code) == "" {
		return apperrors.BadRequest("code is required")
	}
	if in.IssuerType == "" {
		in.IssuerType = domain.IssuerTypePlatform
	}
	if in.IssuerType != domain.IssuerTypePlatform {
		// MVP locks out seller-issued coupons; revisit when Phase 5 lands.
		return apperrors.BadRequest("only platform-issued coupons are supported in MVP").WithCode("UNSUPPORTED_ISSUER")
	}
	switch in.DiscountType {
	case domain.DiscountTypePercent:
		if in.DiscountPercentBps == nil || *in.DiscountPercentBps <= 0 || *in.DiscountPercentBps > 10000 {
			return apperrors.BadRequest("discount_percent_bps must be in (0, 10000]")
		}
		if in.DiscountAmount != nil {
			return apperrors.BadRequest("discount_amount must be null for percent coupons")
		}
	case domain.DiscountTypeFixedAmount:
		if in.DiscountAmount == nil || *in.DiscountAmount <= 0 {
			return apperrors.BadRequest("discount_amount must be positive")
		}
		if in.DiscountPercentBps != nil {
			return apperrors.BadRequest("discount_percent_bps must be null for fixed_amount coupons")
		}
	default:
		return apperrors.BadRequest("discount_type must be 'percent' or 'fixed_amount'")
	}
	if in.MinOrderAmount < 0 {
		return apperrors.BadRequest("min_order_amount must be non-negative")
	}
	if in.UsageLimitTotal != nil && *in.UsageLimitTotal <= 0 {
		return apperrors.BadRequest("usage_limit_total must be positive when set")
	}
	if in.UsageLimitPerUser != nil && *in.UsageLimitPerUser <= 0 {
		return apperrors.BadRequest("usage_limit_per_user must be positive when set")
	}
	return nil
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func clampPage(limit, offset int) (int, int) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
