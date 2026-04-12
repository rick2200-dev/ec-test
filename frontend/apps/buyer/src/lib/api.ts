import { fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";
import type {
  CancellationRequest,
  Inquiry,
  InquiryListResponse,
  InquiryMessage,
  InquiryWithMessages,
  ProductRating,
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
  params: { limit?: number; offset?: number } = {},
): Promise<InquiryListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(
    `/api/v1/buyer/inquiries${qs.toString() ? `?${qs}` : ""}`,
  );
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
export async function postBuyerInquiryMessage(
  id: string,
  body: string,
): Promise<InquiryMessage> {
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
  reason: string,
): Promise<CancellationRequest> {
  const res = await fetchAPI(
    `/api/v1/buyer/orders/${orderId}/cancellation-request`,
    {
      method: "POST",
      body: JSON.stringify({ reason }),
    },
  );
  return jsonOrThrow<CancellationRequest>(res);
}

/**
 * Fetch the latest cancellation request for an order. Returns null when
 * no request has ever been opened (backend returns 404 in that case,
 * which we translate to null so callers can branch cleanly).
 */
export async function getOrderCancellationRequest(
  orderId: string,
): Promise<CancellationRequest | null> {
  const res = await fetchAPI(
    `/api/v1/buyer/orders/${orderId}/cancellation-request`,
  );
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
  params: { limit?: number; offset?: number } = {},
): Promise<ReviewListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(
    `/api/v1/buyer/products/${productId}/reviews${qs.toString() ? `?${qs}` : ""}`,
  );
  return jsonOrThrow<ReviewListResponse>(res);
}

/** Get the aggregate rating for a product. */
export async function getProductRating(
  productId: string,
): Promise<ProductRating> {
  const res = await fetchAPI(
    `/api/v1/buyer/products/${productId}/rating`,
  );
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
  input: { rating?: number; title?: string; body?: string },
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
