import { fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";
import type {
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
