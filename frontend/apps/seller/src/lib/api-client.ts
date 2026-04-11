// Seller API client. Thin fetch wrapper over the gateway's
// /api/v1/seller/* surface. Follows the same shape as the buyer app's
// lib/api.ts to keep auth plumbing consistent across apps once the
// Auth0 session wiring lands.
//
// NOTE: this file intentionally does NOT inject an Authorization header
// yet — the seller app does not have an Auth0 session context mounted.
// Once it does, add a single `headers.Authorization = `Bearer ${token}``
// here. The gateway will then accept either the session JWT or an
// `sk_live_*` API token transparently.

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

// APIError carries the HTTP status and server-side error message so the
// UI can differentiate (e.g. 401 → "session expired", 429 → "rate
// limited", 5xx → "try again later"). Every client method throws this
// on a non-2xx response.
export class APIError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "APIError";
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
  });
  if (!res.ok) {
    // The gateway returns {"error": "..."} on every error path; fall
    // back to the raw status text if the body isn't valid JSON.
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new APIError(res.status, body.error ?? res.statusText);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

// APIToken is the subset of the server-side SellerAPIToken shape the UI
// cares about. Keep in sync with
// backend/services/auth/internal/domain/api_token.go. TokenHash is
// intentionally absent — the server never sends it.
export interface APIToken {
  id: string;
  name: string;
  scopes: string[];
  token_prefix: string;
  token_lookup: string;
  rate_limit_rps: number | null;
  rate_limit_burst: number | null;
  issued_by_auth0_user_id: string;
  expires_at: string | null;
  revoked_at: string | null;
  last_used_at: string | null;
  created_at: string;
  updated_at: string;
}

// CreateAPITokenRequest mirrors issueAPITokenRequest on the auth
// service. ExpiresAt is ISO-8601; null/undefined means "never expires".
export interface CreateAPITokenRequest {
  name: string;
  scopes: string[];
  expires_at?: string | null;
  rate_limit_rps?: number | null;
  rate_limit_burst?: number | null;
}

// CreateAPITokenResponse embeds the persisted token and the one-time
// plaintext string. The UI MUST show the plaintext once and then drop
// it — there is no "reveal again" endpoint.
export interface CreateAPITokenResponse extends APIToken {
  token: string;
}

interface ListAPITokensResponse {
  items: APIToken[];
  total: number;
  limit: number;
  offset: number;
}

// listAPITokens returns the seller's tokens newest-first. Revoked and
// expired rows are included so the UI can render a complete history.
export async function listAPITokens(): Promise<APIToken[]> {
  const resp = await request<ListAPITokensResponse>("/api/v1/seller/api-tokens");
  return resp.items;
}

export async function createAPIToken(
  req: CreateAPITokenRequest,
): Promise<CreateAPITokenResponse> {
  return request<CreateAPITokenResponse>("/api/v1/seller/api-tokens", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function revokeAPIToken(id: string): Promise<void> {
  await request<{ status: string }>(`/api/v1/seller/api-tokens/${id}`, {
    method: "DELETE",
  });
}
