import { fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";
import type {
  CancellationRequest,
  CancellationRequestListResponse,
  CancellationRequestStatus,
  Coupon,
  CouponListResponse,
  CouponStats,
  CreateCouponRequest,
  Inquiry,
  InquiryListResponse,
  InquiryMessage,
  InquiryWithMessages,
  MarkDeliveredRequest,
  RegisterShipmentRequest,
  Review,
  ReviewListResponse,
  ReviewReply,
  Shipment,
  ShipmentListResponse,
  ShipmentStatus,
} from "@ec-marketplace/types";

export { fetchAPI, ApiError } from "@ec-marketplace/api-client";

/** List inquiry threads for the current seller. */
export async function listSellerInquiries(
  params: { limit?: number; offset?: number; status?: "open" | "closed" } = {}
): Promise<InquiryListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  if (params.status) qs.set("status", params.status);
  const res = await fetchAPI(`/api/v1/seller/inquiries${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<InquiryListResponse>(res);
}

export async function getSellerInquiry(id: string): Promise<InquiryWithMessages> {
  const res = await fetchAPI(`/api/v1/seller/inquiries/${id}`);
  return jsonOrThrow<InquiryWithMessages>(res);
}

export async function postSellerInquiryMessage(id: string, body: string): Promise<InquiryMessage> {
  const res = await fetchAPI(`/api/v1/seller/inquiries/${id}/messages`, {
    method: "POST",
    body: JSON.stringify({ body }),
  });
  return jsonOrThrow<InquiryMessage>(res);
}

export async function markSellerInquiryRead(id: string): Promise<Inquiry> {
  const res = await fetchAPI(`/api/v1/seller/inquiries/${id}/read`, {
    method: "POST",
  });
  return jsonOrThrow<Inquiry>(res);
}

export async function closeSellerInquiry(id: string): Promise<Inquiry> {
  const res = await fetchAPI(`/api/v1/seller/inquiries/${id}/close`, {
    method: "POST",
  });
  return jsonOrThrow<Inquiry>(res);
}

/** List cancellation requests for the current seller's tenant. */
export async function listSellerCancellationRequests(
  params: {
    status?: CancellationRequestStatus;
    limit?: number;
    offset?: number;
  } = {}
): Promise<CancellationRequestListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(
    `/api/v1/seller/cancellation-requests${qs.toString() ? `?${qs}` : ""}`
  );
  return jsonOrThrow<CancellationRequestListResponse>(res);
}

/** Fetch a single cancellation request by id. */
export async function getSellerCancellationRequest(id: string): Promise<CancellationRequest> {
  const res = await fetchAPI(`/api/v1/seller/cancellation-requests/${id}`);
  return jsonOrThrow<CancellationRequest>(res);
}

/**
 * Approve a pending cancellation request. Backend orchestrates Stripe
 * refund + transfer reversal + DB updates; on success the returned
 * request is in the `approved` state, on Stripe failure it is in the
 * `failed` state and the ApiError body carries a semantic error code.
 */
export async function approveCancellationRequest(
  id: string,
  sellerComment?: string
): Promise<CancellationRequest> {
  const res = await fetchAPI(`/api/v1/seller/cancellation-requests/${id}/approve`, {
    method: "POST",
    body: JSON.stringify({ seller_comment: sellerComment ?? "" }),
  });
  return jsonOrThrow<CancellationRequest>(res);
}

/**
 * Reject a pending cancellation request. `sellerComment` is required
 * (the backend rejects empty comments with 400).
 */
export async function rejectCancellationRequest(
  id: string,
  sellerComment: string
): Promise<CancellationRequest> {
  const res = await fetchAPI(`/api/v1/seller/cancellation-requests/${id}/reject`, {
    method: "POST",
    body: JSON.stringify({ seller_comment: sellerComment }),
  });
  return jsonOrThrow<CancellationRequest>(res);
}

// ---------------------------------------------------------------------------
// Shipments (seller-side)
// ---------------------------------------------------------------------------

/** List shipments owned by the current seller, optionally filtered by status. */
export async function listSellerShipments(
  params: { status?: ShipmentStatus | "all"; limit?: number; offset?: number } = {}
): Promise<ShipmentListResponse> {
  const qs = new URLSearchParams();
  if (params.status && params.status !== "all") qs.set("status", params.status);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/seller/shipments${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<ShipmentListResponse>(res);
}

/** Get a single shipment by id. */
export async function getSellerShipment(id: string): Promise<Shipment> {
  const res = await fetchAPI(`/api/v1/seller/shipments/${id}`);
  return jsonOrThrow<Shipment>(res);
}

/**
 * Register tracking on a ready_to_ship shipment. Returns the updated
 * shipment in the `shipped` state.
 */
export async function registerShipment(
  id: string,
  body: RegisterShipmentRequest
): Promise<Shipment> {
  const res = await fetchAPI(`/api/v1/seller/shipments/${id}/register`, {
    method: "POST",
    body: JSON.stringify(body),
  });
  return jsonOrThrow<Shipment>(res);
}

/** Mark a shipped shipment as delivered. */
export async function markShipmentDelivered(
  id: string,
  body: MarkDeliveredRequest = {}
): Promise<Shipment> {
  const res = await fetchAPI(`/api/v1/seller/shipments/${id}/deliver`, {
    method: "POST",
    body: JSON.stringify(body),
  });
  return jsonOrThrow<Shipment>(res);
}

// ---------------------------------------------------------------------------
// Reviews (seller-side)
// ---------------------------------------------------------------------------

/** List reviews for the authenticated seller's products. */
export async function listSellerReviews(
  params: { limit?: number; offset?: number } = {}
): Promise<ReviewListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/seller/reviews${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<ReviewListResponse>(res);
}

/** Fetch a single review owned by the current seller. */
export async function getSellerReview(id: string): Promise<Review> {
  const res = await fetchAPI(`/api/v1/seller/reviews/${id}`);
  return jsonOrThrow<Review>(res);
}

/** Post a new reply to a review. 409 if a reply already exists. */
export async function createReviewReply(id: string, body: string): Promise<ReviewReply> {
  const res = await fetchAPI(`/api/v1/seller/reviews/${id}/reply`, {
    method: "POST",
    body: JSON.stringify({ body }),
  });
  return jsonOrThrow<ReviewReply>(res);
}

/** Edit the seller's existing reply. 404 if no reply has been posted. */
export async function updateReviewReply(id: string, body: string): Promise<ReviewReply> {
  const res = await fetchAPI(`/api/v1/seller/reviews/${id}/reply`, {
    method: "PUT",
    body: JSON.stringify({ body }),
  });
  return jsonOrThrow<ReviewReply>(res);
}

/** Delete the seller's reply. */
export async function deleteReviewReply(id: string): Promise<void> {
  const res = await fetchAPI(`/api/v1/seller/reviews/${id}/reply`, {
    method: "DELETE",
  });
  await jsonOrThrow<void>(res);
}

// ---------------------------------------------------------------------------
// Subscription (seller-side)
// ---------------------------------------------------------------------------

import type { SellerSubscription, SubscriptionPlan } from "./types";

export async function listSellerPlans(): Promise<SubscriptionPlan[]> {
  const res = await fetchAPI(`/api/v1/seller/plans`);
  return jsonOrThrow<SubscriptionPlan[]>(res);
}

/** Returns null when the seller has no active subscription (404). */
export async function getMySubscription(): Promise<SellerSubscription | null> {
  const res = await fetchAPI(`/api/v1/seller/subscription`);
  if (res.status === 404) return null;
  return jsonOrThrow<SellerSubscription>(res);
}

export async function subscribeToPlan(input: {
  plan_id: string;
  stripe_payment_method_id?: string;
}): Promise<SellerSubscription> {
  const res = await fetchAPI(`/api/v1/seller/subscription`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<SellerSubscription>(res);
}

// ---------------------------------------------------------------------------
// Coupons (seller-issued)
// ---------------------------------------------------------------------------

export async function listSellerCoupons(
  params: { status?: string; limit?: number; offset?: number } = {}
): Promise<CouponListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/seller/coupons${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<CouponListResponse>(res);
}

export async function getSellerCoupon(id: string): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/seller/coupons/${id}`);
  return jsonOrThrow<Coupon>(res);
}

export async function createSellerCoupon(input: CreateCouponRequest): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/seller/coupons`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<Coupon>(res);
}

/**
 * Editable subset of a seller's own coupon. Server-side guards
 * ensure a seller can only edit their own rows; this client mirrors
 * the admin PUT shape exactly so the backend handler can be shared.
 */
export interface UpdateSellerCouponInput {
  title: string;
  description: string;
  min_order_amount: number;
  max_discount_amount: number | null;
  expires_at_unix: number | null;
  usage_limit_total: number | null;
  usage_limit_per_user: number | null;
}

export async function updateSellerCoupon(
  id: string,
  input: UpdateSellerCouponInput
): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/seller/coupons/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<Coupon>(res);
}

export async function revokeSellerCoupon(id: string): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/seller/coupons/${id}/revoke`, {
    method: "POST",
  });
  return jsonOrThrow<Coupon>(res);
}

export async function getSellerCouponStats(id: string): Promise<CouponStats> {
  const res = await fetchAPI(`/api/v1/seller/coupons/${id}/stats`);
  return jsonOrThrow<CouponStats>(res);
}
