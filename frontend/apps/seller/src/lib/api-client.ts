// Seller API client. Thin wrapper over the gateway's
// /api/v1/seller/* surface. Built on top of the shared
// `@ec-marketplace/api-client` primitives so auth plumbing and error
// handling stay consistent across apps once the Auth0 session wiring
// lands.
//
// NOTE: this module intentionally does NOT inject an Authorization
// header yet — the seller app does not have an Auth0 session context
// mounted. Once it does, add a default header to the shared
// `fetchAPI` (or a wrapper around it) and the gateway will accept
// either the session JWT or an `sk_live_*` API token transparently.

import { ApiError, fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";

// Existing seller code (notably
// `src/app/(dashboard)/settings/api-tokens/page.tsx`) imports the
// error class as `APIError`. Re-export the shared `ApiError` under
// both spellings so call sites keep working without a mass rename.
export { ApiError, ApiError as APIError };

// APIToken is the subset of the server-side SellerAPIToken shape the
// UI cares about. Keep in sync with
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

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetchAPI(path, options);
  return jsonOrThrow<T>(res);
}

// listAPITokens returns the seller's tokens newest-first. Revoked and
// expired rows are included so the UI can render a complete history.
export async function listAPITokens(): Promise<APIToken[]> {
  const resp = await request<ListAPITokensResponse>("/api/v1/seller/api-tokens");
  return resp.items;
}

export async function createAPIToken(req: CreateAPITokenRequest): Promise<CreateAPITokenResponse> {
  return request<CreateAPITokenResponse>("/api/v1/seller/api-tokens", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function revokeAPIToken(id: string): Promise<void> {
  await request<void>(`/api/v1/seller/api-tokens/${id}`, {
    method: "DELETE",
  });
}
