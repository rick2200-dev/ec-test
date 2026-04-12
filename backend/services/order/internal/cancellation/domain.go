package cancellation

import (
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// Status values for an order cancellation request. Mirrors the CHECK
// constraint in migration 000016. Kept as a typed string so handlers,
// events, and tests share one source of truth.
type Status string

const (
	// StatusPending — the buyer has opened a request but the seller has
	// not decided yet. The partial unique index `ux_cancellation_pending_per_order`
	// guarantees at most one request in this state per order.
	StatusPending Status = "pending"
	// StatusApproved — the seller approved and Stripe refund + transfer
	// reversals + DB writes all succeeded. Terminal.
	StatusApproved Status = "approved"
	// StatusRejected — the seller rejected the request. Terminal.
	StatusRejected Status = "rejected"
	// StatusFailed — the seller approved, but the Stripe refund or a
	// transfer reversal failed irrecoverably. The row is kept for audit
	// and so that the partial unique index frees up (allowing a retry
	// after an operator reconciles). Terminal.
	StatusFailed Status = "failed"
)

// CancellationRequest is the domain object for an order cancellation
// request. One request is created per buyer click, and at most one
// request per order is in the `pending` state at any given moment.
type CancellationRequest struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	OrderID              uuid.UUID  `json:"order_id"`
	RequestedByAuth0ID   string     `json:"requested_by_auth0_id"`
	Reason               string     `json:"reason"`
	Status               Status     `json:"status"`
	SellerComment        *string    `json:"seller_comment,omitempty"`
	ProcessedBySellerID  *uuid.UUID `json:"processed_by_seller_id,omitempty"`
	ProcessedAt          *time.Time `json:"processed_at,omitempty"`
	StripeRefundID       *string    `json:"stripe_refund_id,omitempty"`
	FailureReason        *string    `json:"failure_reason,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// canOrderBeCancelled reports whether an order in the given status can
// still accept a cancellation request. The allowed statuses are chosen
// to mirror the product decision: once the order has shipped, a
// cancellation request no longer makes sense and the buyer should use
// a return flow instead (out of scope).
//
// This is a pure function on purpose — it is called twice during the
// approval flow (once at request creation, once again inside Tx1 of the
// approval orchestration) to close the buyer-cancel / seller-ship race.
func canOrderBeCancelled(status string) bool {
	switch status {
	case domain.StatusPending, domain.StatusPaid, domain.StatusProcessing:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether a request has reached a terminal state.
// Used by the service layer to reject re-processing of an already-
// resolved request.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusApproved, StatusRejected, StatusFailed:
		return true
	default:
		return false
	}
}
