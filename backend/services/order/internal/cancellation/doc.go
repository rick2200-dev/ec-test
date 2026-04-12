// Package cancellation implements the order-cancellation-request bounded
// context inside the order service. A buyer opens a cancellation request
// against an order they already paid for; the seller either approves or
// rejects it. Approval orchestrates a Stripe refund, per-payout transfer
// reversals, inventory release (via Pub/Sub), and buyer notification.
//
// # Why this lives inside the order service
//
// Cancellation approval writes to `orders`, `payouts`, and
// `order_cancellation_requests` inside a single database transaction
// (see Service.ApproveCancellation for the two-phase flow). Splitting
// this into a separate microservice would force a Saga over three
// tables owned by the order service, and partial-failure recovery would
// get dramatically harder. The order service already bundles multiple
// sub-domains (orders, payouts, commissions, Stripe), so cancellation
// is kept in the same deployment unit.
//
// # Package boundary — the bounded-context guardrail
//
// This package is the ONLY place that owns the cancellation domain.
// Enforced conventions:
//
//   - This package may import: domain, repository, stripe, pkg/pubsub,
//     pkg/errors, pkg/tenant, pkg/httputil, pkg/database.
//   - The reverse direction is NOT allowed: no file under service/,
//     handler/, grpcserver/, repository/, or stripe/ may import this
//     package. Ownership checks, state machine logic, error codes, and
//     Stripe orchestration all live here.
//   - cmd/server/main.go wires the cancellation service and mounts its
//     HTTP/gRPC handlers in parallel with the existing order handlers.
//
// # Future extraction path
//
// If cancellation ever grows into a full returns/disputes/chargeback
// workflow — or if write volume outgrows the order service — this
// package can be lifted into services/cancellation/ with minimal churn:
// the repository/stripe/pubsub dependencies are already behind
// unexported interfaces declared in service.go, so they can be swapped
// for gRPC clients without touching business logic.
package cancellation
