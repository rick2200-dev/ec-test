package cancellation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// OrderReader is the narrow view of the existing order repository that
// cancellation needs: fetch a single order with its lines. Declared as
// an interface so service_test.go can mock it without reaching into
// pgx or the repository package.
type OrderReader interface {
	GetByID(ctx context.Context, tenantID, orderID uuid.UUID) (*domain.OrderWithLines, error)
}

// PayoutReader is the narrow view of the existing payout repository
// that cancellation needs. Same rationale as OrderReader.
type PayoutReader interface {
	ListByOrderID(ctx context.Context, tenantID, orderID uuid.UUID) ([]domain.Payout, error)
}

// StripeClient is the narrow view of the Stripe wrapper that
// cancellation needs. CreateRefund / ReverseTransfer both return the
// remote id so we can persist it on the request / payout rows.
type StripeClient interface {
	CreateRefund(paymentIntentID string, amount int64, idempotencyKey string) (refundID string, err error)
	ReverseTransfer(transferID string, amount int64, idempotencyKey string) (reversalID string, err error)
}

// RequestsStore is the narrow view of the cancellation repository that
// Service needs. Declared as an interface so service_test.go can stub
// persistence without a real Postgres, while *Repository is the
// production implementation wired from main.go.
type RequestsStore interface {
	Create(ctx context.Context, tenantID uuid.UUID, req *CancellationRequest) error
	GetByID(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error)
	GetLatestByOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*CancellationRequest, error)
	// ListByStatus is seller-scoped — the sellerID parameter is mandatory
	// and must join through order_svc.orders so one seller cannot read
	// another seller's cancellation reasons or buyer identifiers in a
	// multi-seller tenant.
	ListByStatus(ctx context.Context, tenantID, sellerID uuid.UUID, status Status, limit, offset int) ([]CancellationRequest, int, error)
	Reject(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error)
	MarkFailed(ctx context.Context, tenantID, requestID uuid.UUID, failureReason string) error
	ApproveTx(ctx context.Context, tenantID uuid.UUID, in ApprovalTxInput) (*CancellationRequest, *domain.Order, error)
}

// Service orchestrates the order-cancellation workflow end-to-end:
// request creation, seller approval with Stripe refund + transfer
// reversals + DB writes, seller rejection, and downstream event
// publication. All dependencies are interfaces so tests can stub
// both the database and Stripe.
type Service struct {
	orders   OrderReader
	payouts  PayoutReader
	requests RequestsStore
	stripe   StripeClient
	pub      pubsub.Publisher
}

// NewService constructs a Service. Callers wire this in main.go from
// the existing order/payout repositories and Stripe client.
func NewService(
	orders OrderReader,
	payouts PayoutReader,
	requests RequestsStore,
	stripe StripeClient,
	pub pubsub.Publisher,
) *Service {
	return &Service{
		orders:   orders,
		payouts:  payouts,
		requests: requests,
		stripe:   stripe,
		pub:      pub,
	}
}

// RequestCancellation opens a new cancellation request on behalf of
// the given buyer. Ownership, status, and concurrency checks all
// happen here; the repository layer is only responsible for the
// INSERT (plus partial-unique-index detection).
//
// Errors:
//   - NotFound if the order does not exist in the caller's tenant
//   - NotFound + CodeNotOrderBuyer if the caller is not the buyer
//     (wrapped in 404 instead of 403 to avoid leaking existence)
//   - Conflict + CodeOrderNotCancellable if the order is past the
//     cancellable window (shipped/delivered/completed/cancelled)
//   - Conflict + CodeCancellationRequestAlreadyExists if a pending
//     request already exists for this order
func (s *Service) RequestCancellation(ctx context.Context, tenantID, orderID uuid.UUID, buyerAuth0ID, reason string) (*CancellationRequest, error) {
	if reason == "" {
		return nil, apperrors.BadRequest("cancellation reason is required")
	}

	order, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, apperrors.Internal("failed to load order", err)
	}
	if order == nil {
		return nil, apperrors.NotFound("order not found")
	}
	if err := assertBuyerOwnsOrder(&order.Order, buyerAuth0ID); err != nil {
		return nil, err
	}
	if !order.CanBeCancelled() {
		return nil, apperrors.Conflict("order cannot be cancelled in its current status").
			WithCode(CodeOrderNotCancellable)
	}

	req := &CancellationRequest{
		OrderID:            orderID,
		RequestedByAuth0ID: buyerAuth0ID,
		Reason:             reason,
	}
	if err := s.requests.Create(ctx, tenantID, req); err != nil {
		if errors.Is(err, ErrPendingRequestExists) {
			return nil, apperrors.Conflict("a cancellation request is already pending for this order").
				WithCode(CodeCancellationRequestAlreadyExists)
		}
		return nil, apperrors.Internal("failed to create cancellation request", err)
	}

	publishRequested(ctx, s.pub, req, &order.Order)
	return req, nil
}

// GetByID loads a request by id, enforcing tenant scoping through
// the repository's TenantTx. Intended for internal / admin use — the
// seller-facing REST handler must call GetByIDForSeller so it also
// verifies seller ownership of the underlying order.
func (s *Service) GetByID(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
	req, err := s.requests.GetByID(ctx, tenantID, requestID)
	if err != nil {
		return nil, apperrors.Internal("failed to load cancellation request", err)
	}
	if req == nil {
		return nil, apperrors.NotFound("cancellation request not found").
			WithCode(CodeCancellationRequestNotFound)
	}
	return req, nil
}

// GetByIDForSeller loads a request by id AND verifies that the caller is
// the seller of the underlying order. Wraps mismatches in 404 to avoid
// leaking the existence of another seller's request — same pattern as
// assertSellerOwnsOrder used by Approve / Reject. This is what the
// seller-facing REST GET /cancellation-requests/{id} endpoint MUST call
// instead of GetByID, otherwise a multi-seller tenant could read another
// seller's cancellation reason or buyer auth0 id.
func (s *Service) GetByIDForSeller(ctx context.Context, tenantID, sellerID, requestID uuid.UUID) (*CancellationRequest, error) {
	req, _, err := s.loadRequestForSellerAction(ctx, tenantID, requestID, sellerID)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// GetLatestForOrder returns the most recent request for an order (may
// be nil if none has been opened), after verifying the buyer owns the
// order. Used by the buyer order-detail page to render the current
// cancellation state.
func (s *Service) GetLatestForOrder(ctx context.Context, tenantID, orderID uuid.UUID, buyerAuth0ID string) (*CancellationRequest, error) {
	order, err := s.orders.GetByID(ctx, tenantID, orderID)
	if err != nil {
		return nil, apperrors.Internal("failed to load order", err)
	}
	if order == nil {
		return nil, apperrors.NotFound("order not found")
	}
	if err := assertBuyerOwnsOrder(&order.Order, buyerAuth0ID); err != nil {
		return nil, err
	}
	req, err := s.requests.GetLatestByOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, apperrors.Internal("failed to load cancellation request", err)
	}
	return req, nil
}

// ListByStatus returns paginated requests filtered by status AND by
// the calling seller. The sellerID is mandatory and is propagated to
// the repository's JOIN against order_svc.orders so a multi-seller
// tenant cannot read another seller's cancellation reasons or buyer
// auth0 ids via this endpoint.
func (s *Service) ListByStatus(ctx context.Context, tenantID, sellerID uuid.UUID, status Status, limit, offset int) ([]CancellationRequest, int, error) {
	requests, total, err := s.requests.ListByStatus(ctx, tenantID, sellerID, status, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list cancellation requests", err)
	}
	return requests, total, nil
}

// RejectCancellation marks a pending request as rejected with the
// given seller comment. The seller_comment is REQUIRED at the handler
// layer so buyers always have a reason; this method does not enforce
// it a second time.
//
// Errors:
//   - NotFound + CodeCancellationRequestNotFound if the id is unknown
//   - NotFound + CodeNotOrderSeller if the caller is not the seller
//   - Conflict + CodeCancellationRequestAlreadyProcessed if the
//     request is no longer pending
func (s *Service) RejectCancellation(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error) {
	req, order, err := s.loadRequestForSellerAction(ctx, tenantID, requestID, sellerID)
	if err != nil {
		return nil, err
	}
	if req.Status != StatusPending {
		return nil, apperrors.Conflict("cancellation request is no longer pending").
			WithCode(CodeCancellationRequestAlreadyProcessed)
	}

	updated, err := s.requests.Reject(ctx, tenantID, requestID, sellerID, comment)
	if err != nil {
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil, apperrors.Conflict("cancellation request is no longer pending").
				WithCode(CodeCancellationRequestAlreadyProcessed)
		}
		return nil, apperrors.Internal("failed to reject cancellation request", err)
	}

	publishRejected(ctx, s.pub, updated, &order.Order)
	return updated, nil
}

// ApproveCancellation orchestrates a full cancellation approval:
// Stripe refund, per-payout transfer reversals, DB writes, and
// downstream events.
//
// The DB transaction is split into two phases separated by the
// Stripe calls (no DB connection is held while talking to Stripe):
//
//  1. Read phase (loadRequestForSellerAction + payouts lookup): fetch
//     the request, order, order lines, and current payouts. Verify
//     seller ownership, pending status, and that the order is still
//     cancellable (re-check closes the buyer-cancel / seller-ship
//     race — if the seller just shipped, the approval fails before
//     any Stripe call).
//  2. Stripe phase (connection-free):
//     - CreateRefund for order.TotalAmount (partial refund against
//       the shared PaymentIntent). Failure → MarkFailed + error.
//     - ReverseTransfer for each completed payout with a transfer id.
//       Failure mid-loop → MarkFailed + error. Refunds that already
//       succeeded stay on Stripe; the deterministic idempotency keys
//       make the failed request safe to retry after operator reconcile.
//  3. Write phase (repository.ApproveTx): single tenant transaction
//     that flips the request, order, and payout rows with WHERE
//     guards so a concurrent writer loses cleanly.
//  4. Post-commit events: cancellation_approved + order.cancelled
//     (the latter carries line snapshots so inventory can release
//     stock without a callback).
//
// Errors surface the most specific semantic code — the buyer-facing
// error UI switches on these.
func (s *Service) ApproveCancellation(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error) {
	// Phase 1 — read and validate.
	req, order, err := s.loadRequestForSellerAction(ctx, tenantID, requestID, sellerID)
	if err != nil {
		return nil, err
	}
	if req.Status != StatusPending {
		return nil, apperrors.Conflict("cancellation request is no longer pending").
			WithCode(CodeCancellationRequestAlreadyProcessed)
	}
	if !order.CanBeCancelled() {
		return nil, apperrors.Conflict("order cannot be cancelled in its current status").
			WithCode(CodeOrderNotCancellable)
	}
	if order.StripePaymentIntentID == nil || *order.StripePaymentIntentID == "" {
		return nil, apperrors.Conflict("order has no stripe payment intent to refund").
			WithCode(CodeOrderNotCancellable)
	}

	payouts, err := s.payouts.ListByOrderID(ctx, tenantID, order.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to load payouts", err)
	}

	// Phase 2 — Stripe refund (deterministic idempotency key).
	refundKey := fmt.Sprintf("cancellation:%s:refund", requestID)
	refundID, err := s.stripe.CreateRefund(*order.StripePaymentIntentID, order.TotalAmount, refundKey)
	if err != nil {
		failureReason := fmt.Sprintf("stripe refund failed: %v", err)
		if markErr := s.requests.MarkFailed(ctx, tenantID, requestID, failureReason); markErr != nil {
			slog.Error("failed to mark cancellation request as failed after refund failure",
				"error", markErr, "request_id", requestID)
		}
		return nil, apperrors.New(502, "stripe refund failed", err).
			WithCode(CodeRefundFailed)
	}

	// Phase 2 (cont) — per-payout transfer reversals.
	var reversed []ReversedPayout
	for _, p := range payouts {
		// Skip payouts that are not yet completed (webhook race) or
		// that have no transfer to reverse. The refund still covers
		// the buyer-facing money; operators can reconcile Stripe
		// side if a transfer later succeeds.
		if p.Status != domain.PayoutStatusCompleted || p.StripeTransferID == nil || *p.StripeTransferID == "" {
			continue
		}
		reverseKey := fmt.Sprintf("cancellation:%s:reverse:%s", requestID, p.ID)
		reversalID, rerr := s.stripe.ReverseTransfer(*p.StripeTransferID, p.Amount, reverseKey)
		if rerr != nil {
			failureReason := fmt.Sprintf("stripe transfer reversal failed for payout %s: %v", p.ID, rerr)
			if markErr := s.requests.MarkFailed(ctx, tenantID, requestID, failureReason); markErr != nil {
				slog.Error("failed to mark cancellation request as failed after reversal failure",
					"error", markErr, "request_id", requestID)
			}
			return nil, apperrors.New(502, "stripe transfer reversal failed", rerr).
				WithCode(CodeTransferReversalFailed)
		}
		reversed = append(reversed, ReversedPayout{
			PayoutID:         p.ID,
			StripeReversalID: reversalID,
		})
	}

	// Phase 3 — write phase (single tenant transaction).
	in := ApprovalTxInput{
		RequestID:         requestID,
		OrderID:           order.ID,
		SellerID:          sellerID,
		Comment:           comment,
		Reason:            req.Reason,
		StripeRefundID:    refundID,
		ProcessedAt:       time.Now().UTC(),
		ReversedPayouts:   reversed,
		CancellableStatus: cancellableStatuses(),
	}
	updatedReq, updatedOrder, err := s.requests.ApproveTx(ctx, tenantID, in)
	if err != nil {
		if errors.Is(err, ErrAlreadyProcessed) {
			return nil, apperrors.Conflict("cancellation request is no longer pending").
				WithCode(CodeCancellationRequestAlreadyProcessed)
		}
		if errors.Is(err, ErrOrderStatusChanged) {
			// The buyer-cancel / seller-ship race lost at commit
			// time. Stripe has already refunded so the request must
			// be moved to `failed` for manual reconciliation.
			failureReason := "order status changed between approval read and write; stripe refund and reversals must be reconciled manually"
			if markErr := s.requests.MarkFailed(ctx, tenantID, requestID, failureReason); markErr != nil {
				slog.Error("failed to mark cancellation request as failed after status race",
					"error", markErr, "request_id", requestID)
			}
			return nil, apperrors.Conflict("order status changed before approval could commit").
				WithCode(CodeOrderNotCancellable)
		}
		return nil, apperrors.Internal("failed to commit approval", err)
	}

	// Phase 4 — downstream events.
	publishApproved(ctx, s.pub, updatedReq, updatedOrder, order.TotalAmount)
	publishOrderCancelled(ctx, s.pub, updatedReq, updatedOrder, order.Lines)

	return updatedReq, nil
}

// loadRequestForSellerAction loads a request + its order and enforces
// that the calling seller owns the order. Shared by Approve / Reject
// so the ownership check can never drift between the two paths.
// Returns apperrors directly so handlers can pass them through.
func (s *Service) loadRequestForSellerAction(ctx context.Context, tenantID, requestID, sellerID uuid.UUID) (*CancellationRequest, *domain.OrderWithLines, error) {
	req, err := s.requests.GetByID(ctx, tenantID, requestID)
	if err != nil {
		return nil, nil, apperrors.Internal("failed to load cancellation request", err)
	}
	if req == nil {
		return nil, nil, apperrors.NotFound("cancellation request not found").
			WithCode(CodeCancellationRequestNotFound)
	}

	order, err := s.orders.GetByID(ctx, tenantID, req.OrderID)
	if err != nil {
		return nil, nil, apperrors.Internal("failed to load order", err)
	}
	if order == nil {
		// Defensive: FK guarantees this cannot happen, but treat it
		// as a 404 rather than a 500 if it somehow does.
		return nil, nil, apperrors.NotFound("order not found")
	}
	if err := assertSellerOwnsOrder(&order.Order, sellerID); err != nil {
		return nil, nil, err
	}

	return req, order, nil
}

// assertBuyerOwnsOrder checks the buyer matches the order. Mismatches
// are wrapped in 404 so we do not leak the existence of another
// buyer's order — same pattern the existing gateway buyer_handler
// uses for ownership-guarded lookups.
func assertBuyerOwnsOrder(order *domain.Order, buyerAuth0ID string) error {
	if order.BuyerAuth0ID != buyerAuth0ID {
		return apperrors.NotFound("order not found").WithCode(CodeNotOrderBuyer)
	}
	return nil
}

// assertSellerOwnsOrder checks the seller matches the order. Wrapped
// in 404 for the same information-leak reason as assertBuyerOwnsOrder.
func assertSellerOwnsOrder(order *domain.Order, sellerID uuid.UUID) error {
	if order.SellerID != sellerID {
		return apperrors.NotFound("order not found").WithCode(CodeNotOrderSeller)
	}
	return nil
}

// cancellableStatuses returns the list of order statuses for which a
// cancellation approval is still legal. This list must stay in sync with
// domain.Order.CanBeCancelled (the Go-side pre-check). The test
// TestCancellableStatusesMatchesCanOrderBeCancelled enforces alignment
// between these two so they cannot drift silently.
func cancellableStatuses() []string {
	return []string{
		domain.StatusPending,
		domain.StatusPaid,
		domain.StatusProcessing,
	}
}
