import { fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";
import type {
  CancellationRequest,
  CancellationRequestListResponse,
  CancellationRequestStatus,
  Inquiry,
  InquiryListResponse,
  InquiryMessage,
  InquiryWithMessages,
} from "@ec-marketplace/types";

export { fetchAPI, ApiError } from "@ec-marketplace/api-client";

/** List inquiry threads for the current seller. */
export async function listSellerInquiries(
  params: { limit?: number; offset?: number; status?: "open" | "closed" } = {},
): Promise<InquiryListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  if (params.status) qs.set("status", params.status);
  const res = await fetchAPI(
    `/api/v1/seller/inquiries${qs.toString() ? `?${qs}` : ""}`,
  );
  return jsonOrThrow<InquiryListResponse>(res);
}

export async function getSellerInquiry(id: string): Promise<InquiryWithMessages> {
  const res = await fetchAPI(`/api/v1/seller/inquiries/${id}`);
  return jsonOrThrow<InquiryWithMessages>(res);
}

export async function postSellerInquiryMessage(
  id: string,
  body: string,
): Promise<InquiryMessage> {
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
  } = {},
): Promise<CancellationRequestListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(
    `/api/v1/seller/cancellation-requests${qs.toString() ? `?${qs}` : ""}`,
  );
  return jsonOrThrow<CancellationRequestListResponse>(res);
}

/** Fetch a single cancellation request by id. */
export async function getSellerCancellationRequest(
  id: string,
): Promise<CancellationRequest> {
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
  sellerComment?: string,
): Promise<CancellationRequest> {
  const res = await fetchAPI(
    `/api/v1/seller/cancellation-requests/${id}/approve`,
    {
      method: "POST",
      body: JSON.stringify({ seller_comment: sellerComment ?? "" }),
    },
  );
  return jsonOrThrow<CancellationRequest>(res);
}

/**
 * Reject a pending cancellation request. `sellerComment` is required
 * (the backend rejects empty comments with 400).
 */
export async function rejectCancellationRequest(
  id: string,
  sellerComment: string,
): Promise<CancellationRequest> {
  const res = await fetchAPI(
    `/api/v1/seller/cancellation-requests/${id}/reject`,
    {
      method: "POST",
      body: JSON.stringify({ seller_comment: sellerComment }),
    },
  );
  return jsonOrThrow<CancellationRequest>(res);
}
