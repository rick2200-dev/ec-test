package cancellation

// Semantic error codes surfaced by the order-cancellation package.
//
// These values are part of the PUBLIC API contract between the order
// service and its callers (gateway, frontend apps, seller dashboard).
// Clients switch on them to render the right UI message for errors
// that share an HTTP status but mean different things. As with every
// other semantic code in the repo, keep them:
//
//   - SCREAMING_SNAKE_CASE
//   - stable across releases (renaming is a breaking change)
//   - attached via (*AppError).WithCode from pkg/errors
//
// A short description of each code appears in docs/order-cancellation.md
// §API — keep it in sync when adding a new code here.
const (
	// CodeOrderNotCancellable is returned when the buyer tries to open
	// a cancellation request against an order whose status is outside
	// {pending, paid, processing}. Carries HTTP 409 Conflict.
	CodeOrderNotCancellable = "ORDER_NOT_CANCELLABLE"

	// CodeCancellationRequestNotFound is returned when no request row
	// matches the given id within the caller's tenant. Carries HTTP 404.
	CodeCancellationRequestNotFound = "CANCELLATION_REQUEST_NOT_FOUND"

	// CodeCancellationRequestAlreadyExists is returned when the buyer
	// tries to open a second pending request while one is already open.
	// The partial unique index `ux_cancellation_pending_per_order`
	// makes this the authoritative race guard. HTTP 409 Conflict.
	CodeCancellationRequestAlreadyExists = "CANCELLATION_REQUEST_ALREADY_EXISTS"

	// CodeCancellationRequestAlreadyProcessed is returned when the seller
	// tries to approve or reject a request that is no longer in the
	// pending state (someone else already decided, or a prior failure
	// terminated it). HTTP 409 Conflict.
	CodeCancellationRequestAlreadyProcessed = "CANCELLATION_REQUEST_ALREADY_PROCESSED"

	// CodeRefundFailed is returned when the Stripe CreateRefund call
	// fails during approval. The request is transitioned to `failed`
	// and the order stays in its previous status. HTTP 502 Bad Gateway
	// (upstream failure).
	CodeRefundFailed = "REFUND_FAILED"

	// CodeTransferReversalFailed is returned when the Stripe
	// TransferReversal call fails AFTER a successful refund — the
	// canonical partial-failure case. The request is transitioned to
	// `failed` and must be reconciled manually. HTTP 502.
	CodeTransferReversalFailed = "TRANSFER_REVERSAL_FAILED"

	// CodeNotOrderBuyer is returned when the authenticated user is not
	// the buyer of the target order. Wrapped in a 404 (not 403) to
	// avoid leaking order existence across tenants — the buyer_handler
	// ownership-check pattern.
	CodeNotOrderBuyer = "NOT_ORDER_BUYER"

	// CodeNotOrderSeller is returned when the authenticated seller is
	// not the seller of the target order. Same 404-wrapping rule as
	// CodeNotOrderBuyer.
	CodeNotOrderSeller = "NOT_ORDER_SELLER"
)
