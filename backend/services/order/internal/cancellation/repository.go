package cancellation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// pgUniqueViolation is the PostgreSQL SQLSTATE for a unique constraint
// violation (23505). Detected via *pgconn.PgError so the check is
// robust across driver versions and locales.
const pgUniqueViolation = "23505"

// pendingUniqueIndex is the name of the partial unique index that
// guarantees at most one pending cancellation request per order (see
// migration 000016). We match on the index name so we only translate
// THIS specific constraint violation into ErrPendingRequestExists —
// any other 23505 keeps its original error.
const pendingUniqueIndex = "ux_cancellation_pending_per_order"

// ErrPendingRequestExists is the sentinel returned by Create when a
// second pending request is attempted against the same order. The
// service layer maps this to the CodeCancellationRequestAlreadyExists
// semantic error.
var ErrPendingRequestExists = errors.New("pending cancellation request already exists for order")

// Repository persists CancellationRequest rows inside a TenantTx so
// all reads and writes go through the `tenant_isolation` RLS policy
// declared in migration 000016.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a Repository bound to the given pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a new request with status='pending'. Returns
// ErrPendingRequestExists if another pending request already exists
// for the same order (authoritative race guard via the partial unique
// index).
func (r *Repository) Create(ctx context.Context, tenantID uuid.UUID, req *CancellationRequest) error {
	req.ID = uuid.New()
	req.TenantID = tenantID
	req.Status = StatusPending

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO order_svc.order_cancellation_requests
			 (id, tenant_id, order_id, requested_by_auth0_id, reason, status)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING created_at, updated_at`,
			req.ID, req.TenantID, req.OrderID, req.RequestedByAuth0ID, req.Reason, string(req.Status),
		).Scan(&req.CreatedAt, &req.UpdatedAt)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) &&
				pgErr.Code == pgUniqueViolation &&
				pgErr.ConstraintName == pendingUniqueIndex {
				return ErrPendingRequestExists
			}
			return fmt.Errorf("insert cancellation request: %w", err)
		}
		return nil
	})
}

// GetByID loads a request by id. Returns (nil, nil) when no row is
// found so callers can distinguish missing-row from DB error.
func (r *Repository) GetByID(ctx context.Context, tenantID, requestID uuid.UUID) (*CancellationRequest, error) {
	var req CancellationRequest
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return scanOne(ctx, tx, &req, &found,
			`SELECT id, tenant_id, order_id, requested_by_auth0_id, reason, status,
			        seller_comment, processed_by_seller_id, processed_at,
			        stripe_refund_id, failure_reason, created_at, updated_at
			 FROM order_svc.order_cancellation_requests
			 WHERE id = $1 AND tenant_id = $2`,
			requestID, tenantID,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("get cancellation request: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &req, nil
}

// GetLatestByOrder returns the most recent request for an order, or
// (nil, nil) if no request has ever been opened against it. Used by
// the buyer order-detail page to decide whether to show the "Cancel"
// button or the current request status.
func (r *Repository) GetLatestByOrder(ctx context.Context, tenantID, orderID uuid.UUID) (*CancellationRequest, error) {
	var req CancellationRequest
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return scanOne(ctx, tx, &req, &found,
			`SELECT id, tenant_id, order_id, requested_by_auth0_id, reason, status,
			        seller_comment, processed_by_seller_id, processed_at,
			        stripe_refund_id, failure_reason, created_at, updated_at
			 FROM order_svc.order_cancellation_requests
			 WHERE tenant_id = $1 AND order_id = $2
			 ORDER BY created_at DESC
			 LIMIT 1`,
			tenantID, orderID,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("get latest cancellation request: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &req, nil
}

// ListByStatus returns paginated requests filtered by status AND constrained
// to the given seller, by joining against order_svc.orders and matching
// orders.seller_id. The seller_id filter is mandatory — a multi-seller tenant
// must not be able to read another seller's cancellation reasons or buyer
// auth0 ids through this endpoint. The underlying cancellation table has no
// denormalized seller_id column, so we join rather than widen the schema.
func (r *Repository) ListByStatus(ctx context.Context, tenantID uuid.UUID, sellerID uuid.UUID, status Status, limit, offset int) ([]CancellationRequest, int, error) {
	var requests []CancellationRequest
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*)
			 FROM order_svc.order_cancellation_requests cr
			 JOIN order_svc.orders o
			   ON o.id = cr.order_id AND o.tenant_id = cr.tenant_id
			 WHERE cr.tenant_id = $1 AND cr.status = $2 AND o.seller_id = $3`,
			tenantID, string(status), sellerID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count cancellation requests: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT cr.id, cr.tenant_id, cr.order_id, cr.requested_by_auth0_id, cr.reason, cr.status,
			        cr.seller_comment, cr.processed_by_seller_id, cr.processed_at,
			        cr.stripe_refund_id, cr.failure_reason, cr.created_at, cr.updated_at
			 FROM order_svc.order_cancellation_requests cr
			 JOIN order_svc.orders o
			   ON o.id = cr.order_id AND o.tenant_id = cr.tenant_id
			 WHERE cr.tenant_id = $1 AND cr.status = $2 AND o.seller_id = $3
			 ORDER BY cr.created_at DESC
			 LIMIT $4 OFFSET $5`,
			tenantID, string(status), sellerID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list cancellation requests: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var req CancellationRequest
			if err := scanRow(rows, &req); err != nil {
				return err
			}
			requests = append(requests, req)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

// ListByBuyer returns paginated requests opened by the given buyer.
// Used by the buyer "my requests" page if / when we build it.
func (r *Repository) ListByBuyer(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, limit, offset int) ([]CancellationRequest, int, error) {
	var requests []CancellationRequest
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM order_svc.order_cancellation_requests
			 WHERE tenant_id = $1 AND requested_by_auth0_id = $2`,
			tenantID, buyerAuth0ID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count buyer cancellation requests: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, order_id, requested_by_auth0_id, reason, status,
			        seller_comment, processed_by_seller_id, processed_at,
			        stripe_refund_id, failure_reason, created_at, updated_at
			 FROM order_svc.order_cancellation_requests
			 WHERE tenant_id = $1 AND requested_by_auth0_id = $2
			 ORDER BY created_at DESC
			 LIMIT $3 OFFSET $4`,
			tenantID, buyerAuth0ID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list buyer cancellation requests: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var req CancellationRequest
			if err := scanRow(rows, &req); err != nil {
				return err
			}
			requests = append(requests, req)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return requests, total, nil
}

// Reject atomically transitions a request from pending → rejected,
// stamping seller_comment, processed_by_seller_id, and processed_at.
// The WHERE clause re-asserts status='pending' so a concurrent
// approval/rejection is a no-op (RowsAffected == 0) and returns
// ErrAlreadyProcessed. Callers must already have verified that the
// seller owns the order — this method only enforces the state guard.
func (r *Repository) Reject(ctx context.Context, tenantID, requestID, sellerID uuid.UUID, comment string) (*CancellationRequest, error) {
	var updated CancellationRequest
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return scanOne(ctx, tx, &updated, &found,
			`UPDATE order_svc.order_cancellation_requests
			 SET status = 'rejected',
			     seller_comment = $3,
			     processed_by_seller_id = $4,
			     processed_at = NOW(),
			     updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
			 RETURNING id, tenant_id, order_id, requested_by_auth0_id, reason, status,
			           seller_comment, processed_by_seller_id, processed_at,
			           stripe_refund_id, failure_reason, created_at, updated_at`,
			requestID, tenantID, comment, sellerID,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("reject cancellation request: %w", err)
	}
	if !found {
		return nil, ErrAlreadyProcessed
	}
	return &updated, nil
}

// MarkFailed atomically transitions a request from pending → failed,
// recording failure_reason. Used after a Stripe refund or reversal
// irrecoverably errors during approval. Same pending-guard as Reject.
func (r *Repository) MarkFailed(ctx context.Context, tenantID, requestID uuid.UUID, failureReason string) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE order_svc.order_cancellation_requests
			 SET status = 'failed',
			     failure_reason = $3,
			     updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2 AND status = 'pending'`,
			requestID, tenantID, failureReason,
		)
		if err != nil {
			return fmt.Errorf("mark cancellation failed: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrAlreadyProcessed
		}
		return nil
	})
}

// ApprovalTxInput is the bundle passed to ApproveTx — everything the
// final write-phase transaction needs after Stripe has already been
// called successfully.
type ApprovalTxInput struct {
	RequestID         uuid.UUID
	OrderID           uuid.UUID
	SellerID          uuid.UUID
	Comment           string
	Reason            string
	StripeRefundID    string
	ProcessedAt       time.Time
	ReversedPayouts   []ReversedPayout
	CancellableStatus []string // the WHERE status IN (...) guard for the order row
}

// ReversedPayout describes one payout row that has just had its
// transfer reversed via Stripe. ApproveTx writes these back to the DB
// as status='reversed' with the captured Stripe reversal id.
type ReversedPayout struct {
	PayoutID         uuid.UUID
	StripeReversalID string
}

// ApproveTx writes every row affected by a successful cancellation
// approval in one tenant transaction:
//
//   - cancellation_requests row → approved (re-asserting status='pending')
//   - orders row → cancelled (re-asserting the status IN (...) guard so a
//     concurrent shipment wins and the approval bails out)
//   - payouts rows → reversed / reversed_at / stripe_reversal_id
//
// Returns the updated request and order so the caller can publish
// downstream events from the freshly persisted state. If the WHERE
// guards miss (RowsAffected == 0) the whole transaction is rolled back
// and ErrAlreadyProcessed / ErrOrderStatusChanged is surfaced so the
// service layer can translate to an appropriate semantic error.
func (r *Repository) ApproveTx(ctx context.Context, tenantID uuid.UUID, in ApprovalTxInput) (*CancellationRequest, *domain.Order, error) {
	var updatedReq CancellationRequest
	var updatedOrder domain.Order
	var reqFound bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		// 1) Flip the request row. Re-assert status='pending' so we
		// fail fast if another caller already resolved it.
		if err := scanOne(ctx, tx, &updatedReq, &reqFound,
			`UPDATE order_svc.order_cancellation_requests
			 SET status = 'approved',
			     seller_comment = NULLIF($3, ''),
			     processed_by_seller_id = $4,
			     processed_at = $5,
			     stripe_refund_id = $6,
			     updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
			 RETURNING id, tenant_id, order_id, requested_by_auth0_id, reason, status,
			           seller_comment, processed_by_seller_id, processed_at,
			           stripe_refund_id, failure_reason, created_at, updated_at`,
			in.RequestID, tenantID, in.Comment, in.SellerID, in.ProcessedAt, in.StripeRefundID,
		); err != nil {
			return fmt.Errorf("approve cancellation request: %w", err)
		}
		if !reqFound {
			return ErrAlreadyProcessed
		}

		// 2) Flip the order row. Re-assert the allowed status guard so
		// a concurrent shipment takes priority and the approval fails.
		scanErr := tx.QueryRow(ctx,
			`UPDATE order_svc.orders
			 SET status = 'cancelled',
			     cancelled_at = $3,
			     cancellation_reason = $4,
			     updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2
			   AND status = ANY($5)
			 RETURNING id, tenant_id, seller_id, seller_name, buyer_auth0_id, status,
			           subtotal_amount, shipping_fee, commission_amount, total_amount, currency,
			           shipping_address, stripe_payment_intent_id, paid_at, cancelled_at, cancellation_reason,
			           created_at, updated_at`,
			in.OrderID, tenantID, in.ProcessedAt, in.Reason, in.CancellableStatus,
		).Scan(
			&updatedOrder.ID, &updatedOrder.TenantID, &updatedOrder.SellerID, &updatedOrder.SellerName,
			&updatedOrder.BuyerAuth0ID, &updatedOrder.Status,
			&updatedOrder.SubtotalAmount, &updatedOrder.ShippingFee, &updatedOrder.CommissionAmount,
			&updatedOrder.TotalAmount, &updatedOrder.Currency,
			&updatedOrder.ShippingAddress, &updatedOrder.StripePaymentIntentID, &updatedOrder.PaidAt,
			&updatedOrder.CancelledAt, &updatedOrder.CancellationReason,
			&updatedOrder.CreatedAt, &updatedOrder.UpdatedAt,
		)
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return ErrOrderStatusChanged
		}
		if scanErr != nil {
			return fmt.Errorf("cancel order: %w", scanErr)
		}

		// 3) Mark each reversed payout. These rows were selected in
		// the read phase by order_id, so their ids are trusted.
		for _, rp := range in.ReversedPayouts {
			tag, err := tx.Exec(ctx,
				`UPDATE order_svc.payouts
				 SET status = $3,
				     stripe_reversal_id = $4,
				     reversed_at = $5
				 WHERE id = $1 AND tenant_id = $2`,
				rp.PayoutID, tenantID, domain.PayoutStatusReversed, rp.StripeReversalID, in.ProcessedAt,
			)
			if err != nil {
				return fmt.Errorf("mark payout reversed: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("payout %s not found", rp.PayoutID)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &updatedReq, &updatedOrder, nil
}

// ErrAlreadyProcessed is returned by Reject / ApproveTx / MarkFailed
// when a state guard (status='pending') fails — typically because
// another actor resolved the request between the read and the write.
var ErrAlreadyProcessed = errors.New("cancellation request already processed")

// ErrOrderStatusChanged is returned by ApproveTx when the order row
// is no longer in one of the cancellable statuses — the buyer-cancel
// vs seller-ship race. The Stripe refund has already completed by
// this point, so the caller must transition the request to `failed`
// and surface a semantic error for manual reconciliation.
var ErrOrderStatusChanged = errors.New("order status changed before approval write")

// rowScanner matches both pgx.Row and pgx.Rows. Used so scanRow /
// scanOne can be shared across QueryRow and Query loops.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanRow reads one request row from the 13-column SELECT projection
// used by every read method in this file. Kept in one place so the
// column list and the Go field order cannot drift.
func scanRow(row rowScanner, req *CancellationRequest) error {
	var status string
	if err := row.Scan(
		&req.ID, &req.TenantID, &req.OrderID, &req.RequestedByAuth0ID, &req.Reason, &status,
		&req.SellerComment, &req.ProcessedBySellerID, &req.ProcessedAt,
		&req.StripeRefundID, &req.FailureReason, &req.CreatedAt, &req.UpdatedAt,
	); err != nil {
		return fmt.Errorf("scan cancellation request: %w", err)
	}
	req.Status = Status(status)
	return nil
}

// scanOne runs a QueryRow, scans the result into req via scanRow, and
// sets found=true when a row comes back. pgx.ErrNoRows is swallowed
// so callers distinguish missing-row from DB error via the `found`
// out-parameter, matching the pattern used in order_repo.go.
func scanOne(ctx context.Context, tx pgx.Tx, req *CancellationRequest, found *bool, sql string, args ...any) error {
	err := scanRow(tx.QueryRow(ctx, sql, args...), req)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	*found = true
	return nil
}
