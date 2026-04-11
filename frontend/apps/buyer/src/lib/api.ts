import type {
  Inquiry,
  InquiryListResponse,
  InquiryMessage,
  InquiryWithMessages,
} from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function fetchAPI(path: string, options?: RequestInit) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  return res;
}

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

/**
 * ApiError carries the HTTP status code alongside the parsed error body so
 * callers can branch on status (e.g. 403) without string-matching on message
 * content. Always thrown from `jsonOrThrow` on non-2xx responses.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

async function jsonOrThrow<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let detail = "";
    let parsed: unknown = null;
    try {
      parsed = await res.json();
      const body = parsed as { error?: string; message?: string };
      detail = body.error ?? body.message ?? "";
    } catch {
      // ignore parse errors
    }
    throw new ApiError(
      res.status,
      detail || `request failed: ${res.status}`,
      parsed,
    );
  }
  return (await res.json()) as T;
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
