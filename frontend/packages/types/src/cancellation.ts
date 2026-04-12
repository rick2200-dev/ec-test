/**
 * Order cancellation request types mirror the JSON shape of
 * `CancellationRequest` in
 * `backend/services/order/internal/cancellation/domain.go`.
 *
 * Used by both the buyer app (to show request status on the order
 * detail page) and the seller app (for the cancellation-requests
 * dashboard list).
 */

/**
 * Terminal and non-terminal states of a cancellation request. Mirrors
 * the Go `Status` type in the cancellation package and the CHECK
 * constraint in migration 000016.
 */
export type CancellationRequestStatus = "pending" | "approved" | "rejected" | "failed";

/** One order cancellation request row. */
export interface CancellationRequest {
  id: string;
  tenant_id: string;
  order_id: string;
  requested_by_auth0_id: string;
  reason: string;
  status: CancellationRequestStatus;
  seller_comment?: string | null;
  processed_by_seller_id?: string | null;
  processed_at?: string | null;
  stripe_refund_id?: string | null;
  failure_reason?: string | null;
  created_at: string;
  updated_at: string;
}

/** Paginated seller list response (matches `pagination.Response[T]`). */
export interface CancellationRequestListResponse {
  items: CancellationRequest[];
  total: number;
  limit: number;
  offset: number;
}
