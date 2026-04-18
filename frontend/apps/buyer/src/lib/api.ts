import { fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";
import type {
  CancellationRequest,
  Cart,
  CheckoutRequest,
  CheckoutResult,
  CouponPreviewRequest,
  CouponPreviewResult,
  CouponRedemptionListResponse,
  Inquiry,
  InquiryListResponse,
  InquiryMessage,
  InquiryWithMessages,
  PointsBalance,
  PointsTransactionListResponse,
  ProductRating,
  SearchRequestParams,
  SearchResponse,
  SearchSuggestResponse,
  Shipment,
  Review,
  ReviewListResponse,
} from "@ec-marketplace/types";

// Re-export the shared primitives so existing call sites that import
// `fetchAPI` / `ApiError` from this module keep working.
export { fetchAPI, ApiError } from "@ec-marketplace/api-client";

export async function trackEvent(eventType: string, productId: string) {
  try {
    await fetchAPI("/api/v1/buyer/events", {
      method: "POST",
      body: JSON.stringify({ event_type: eventType, product_id: productId }),
    });
  } catch {
    // Silently fail - tracking should not break the UI
  }
}

/** List the current buyer's inquiry threads. */
export async function listBuyerInquiries(
  params: { limit?: number; offset?: number } = {}
): Promise<InquiryListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/buyer/inquiries${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<InquiryListResponse>(res);
}

/** Fetch one inquiry thread with all messages. */
export async function getBuyerInquiry(id: string): Promise<InquiryWithMessages> {
  const res = await fetchAPI(`/api/v1/buyer/inquiries/${id}`);
  return jsonOrThrow<InquiryWithMessages>(res);
}

/** Create a new inquiry thread (backend checks the buyer has purchased the SKU). */
export async function createInquiry(input: {
  seller_id: string;
  sku_id: string;
  subject: string;
  initial_body: string;
}): Promise<InquiryWithMessages> {
  const res = await fetchAPI(`/api/v1/buyer/inquiries`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<InquiryWithMessages>(res);
}

/** Append a buyer reply to an existing thread. */
export async function postBuyerInquiryMessage(id: string, body: string): Promise<InquiryMessage> {
  const res = await fetchAPI(`/api/v1/buyer/inquiries/${id}/messages`, {
    method: "POST",
    body: JSON.stringify({ body }),
  });
  return jsonOrThrow<InquiryMessage>(res);
}

/** Mark all seller messages in a thread as read. */
export async function markBuyerInquiryRead(id: string): Promise<Inquiry> {
  const res = await fetchAPI(`/api/v1/buyer/inquiries/${id}/read`, {
    method: "POST",
  });
  return jsonOrThrow<Inquiry>(res);
}

/**
 * Open a cancellation request against an order. Backend enforces the
 * order status (only pending/paid/processing) and the partial unique
 * index prevents a second pending request from being created.
 */
export async function requestOrderCancellation(
  orderId: string,
  reason: string
): Promise<CancellationRequest> {
  const res = await fetchAPI(`/api/v1/buyer/orders/${orderId}/cancellation-request`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
  return jsonOrThrow<CancellationRequest>(res);
}

/**
 * Fetch the latest cancellation request for an order. Returns null when
 * no request has ever been opened (backend returns 404 in that case,
 * which we translate to null so callers can branch cleanly).
 */
export async function getOrderCancellationRequest(
  orderId: string
): Promise<CancellationRequest | null> {
  const res = await fetchAPI(`/api/v1/buyer/orders/${orderId}/cancellation-request`);
  if (res.status === 404) {
    return null;
  }
  return jsonOrThrow<CancellationRequest>(res);
}

// ---------------------------------------------------------------------------
// Reviews
// ---------------------------------------------------------------------------

/** List reviews for a product with pagination. */
export async function listProductReviews(
  productId: string,
  params: { limit?: number; offset?: number } = {}
): Promise<ReviewListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(
    `/api/v1/buyer/products/${productId}/reviews${qs.toString() ? `?${qs}` : ""}`
  );
  return jsonOrThrow<ReviewListResponse>(res);
}

/** Get the aggregate rating for a product. */
export async function getProductRating(productId: string): Promise<ProductRating> {
  const res = await fetchAPI(`/api/v1/buyer/products/${productId}/rating`);
  return jsonOrThrow<ProductRating>(res);
}

/** Create a review for a product. */
export async function createReview(input: {
  product_id: string;
  rating: number;
  title: string;
  body: string;
}): Promise<Review> {
  const res = await fetchAPI(`/api/v1/buyer/reviews`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<Review>(res);
}

/** Update an existing review. */
export async function updateReview(
  id: string,
  input: { rating?: number; title?: string; body?: string }
): Promise<Review> {
  const res = await fetchAPI(`/api/v1/buyer/reviews/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<Review>(res);
}

/** Delete a review. */
export async function deleteReview(id: string): Promise<void> {
  const res = await fetchAPI(`/api/v1/buyer/reviews/${id}`, {
    method: "DELETE",
  });
  await jsonOrThrow<void>(res);
}

// ---------------------------------------------------------------------------
// Cart
// ---------------------------------------------------------------------------

/** Fetch the current buyer's cart (always 200; empty cart returns items=[]). */
export async function getCart(): Promise<Cart> {
  const res = await fetchAPI(`/api/v1/buyer/cart`);
  return jsonOrThrow<Cart>(res);
}

/** Empty the cart. Returns the (now-empty) cart for UI state symmetry. */
export async function clearCart(): Promise<Cart> {
  const res = await fetchAPI(`/api/v1/buyer/cart`, { method: "DELETE" });
  return jsonOrThrow<Cart>(res);
}

/** Add `quantity` of a SKU to the cart (sums with any existing line). */
export async function addCartItem(skuId: string, quantity: number): Promise<Cart> {
  const res = await fetchAPI(`/api/v1/buyer/cart/items`, {
    method: "POST",
    body: JSON.stringify({ sku_id: skuId, quantity }),
  });
  return jsonOrThrow<Cart>(res);
}

/** Set the absolute quantity for a SKU already in the cart. */
export async function updateCartItem(skuId: string, quantity: number): Promise<Cart> {
  const res = await fetchAPI(`/api/v1/buyer/cart/items/${skuId}`, {
    method: "PUT",
    body: JSON.stringify({ quantity }),
  });
  return jsonOrThrow<Cart>(res);
}

/** Remove a SKU line from the cart entirely. */
export async function removeCartItem(skuId: string): Promise<Cart> {
  const res = await fetchAPI(`/api/v1/buyer/cart/items/${skuId}`, {
    method: "DELETE",
  });
  return jsonOrThrow<Cart>(res);
}

/**
 * Submit the checkout. The backend re-validates prices, applies the
 * optional coupon + points, and returns the order ids + Stripe client
 * secret for any payment-confirmation step.
 */
export async function checkoutCart(input: CheckoutRequest): Promise<CheckoutResult> {
  const res = await fetchAPI(`/api/v1/buyer/cart/checkout`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<CheckoutResult>(res);
}

// ---------------------------------------------------------------------------
// Coupon preview
// ---------------------------------------------------------------------------

/**
 * Dry-run a coupon code against the current cart's per-seller subtotals.
 * 404 with code `COUPON_NOT_FOUND` means "no such code"; other 4xx come
 * back as ApiError with their semantic code so the UI can branch.
 */
export async function previewCoupon(input: CouponPreviewRequest): Promise<CouponPreviewResult> {
  const res = await fetchAPI(`/api/v1/buyer/coupons/preview`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<CouponPreviewResult>(res);
}

// ---------------------------------------------------------------------------
// Loyalty points
// ---------------------------------------------------------------------------

/** Get the current buyer's loyalty points balance. */
export async function getPointsBalance(): Promise<PointsBalance> {
  const res = await fetchAPI(`/api/v1/buyer/points/balance`);
  return jsonOrThrow<PointsBalance>(res);
}

/** Paginated history of point ledger entries for the current buyer. */
export async function listPointsTransactions(
  params: { limit?: number; offset?: number } = {}
): Promise<PointsTransactionListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/buyer/points/transactions${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<PointsTransactionListResponse>(res);
}

/** Paginated coupon redemption history for the current buyer. */
export async function listMyCouponRedemptions(
  params: { limit?: number; offset?: number } = {}
): Promise<CouponRedemptionListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/buyer/coupons/redemptions${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<CouponRedemptionListResponse>(res);
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

/** Full-text + faceted product search. */
export async function searchProducts(params: SearchRequestParams): Promise<SearchResponse> {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.seller_id) qs.set("seller_id", params.seller_id);
  if (params.category_id) qs.set("category_id", params.category_id);
  if (params.min_price != null) qs.set("min_price", String(params.min_price));
  if (params.max_price != null) qs.set("max_price", String(params.max_price));
  if (params.status) qs.set("status", params.status);
  if (params.sort_by) qs.set("sort_by", params.sort_by);
  if (params.sort_order) qs.set("sort_order", params.sort_order);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(`/api/v1/buyer/search${qs.toString() ? `?${qs}` : ""}`);
  return jsonOrThrow<SearchResponse>(res);
}

/** Lightweight typeahead suggestions endpoint. */
export async function searchSuggest(
  q: string,
  options?: { signal?: AbortSignal }
): Promise<SearchSuggestResponse> {
  const qs = new URLSearchParams();
  qs.set("q", q);
  const res = await fetchAPI(`/api/v1/buyer/search/suggest?${qs.toString()}`, {
    signal: options?.signal,
  });
  return jsonOrThrow<SearchSuggestResponse>(res);
}

// ---------------------------------------------------------------------------
// Shipment (buyer-side, scoped to an order)
// ---------------------------------------------------------------------------

/**
 * Fetch the shipment for an order. Returns null on 404 so the UI can
 * render a "preparing" state when tracking hasn't been registered yet.
 */
export async function getOrderShipment(orderId: string): Promise<Shipment | null> {
  const res = await fetchAPI(`/api/v1/buyer/orders/${encodeURIComponent(orderId)}/shipment`);
  if (res.status === 404) return null;
  return jsonOrThrow<Shipment>(res);
}
