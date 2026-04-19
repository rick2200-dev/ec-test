/**
 * Admin app API wrappers. Built on `@ec-marketplace/api-client` so the
 * shape mirrors `apps/buyer/src/lib/api.ts` and
 * `apps/seller/src/lib/api.ts`. Per-feature helpers are added as the
 * corresponding admin pages get wired up.
 *
 * NOTE: the admin app does not yet have auth wired up. Every call made
 * by this module arrives at the gateway without an `Authorization`
 * header, which the gateway will reject with 401. The list / get
 * helpers therefore tolerate failure and return an empty shape so the
 * admin pages can render (empty) while auth is still pending. Once
 * admin Auth0 lands, switch to strict `jsonOrThrow` behavior.
 */

import { fetchAPI, jsonOrThrow } from "@ec-marketplace/api-client";
import type {
  AdjustPointsRequest,
  AdjustPointsResponse,
  Coupon,
  CouponListResponse,
  CouponStats,
  CreateCouponRequest,
  PointsBalance,
  PointsTransactionListResponse,
} from "@ec-marketplace/types";
import type { CommissionRule, Seller, SubscriptionPlan } from "./types";

export { fetchAPI, ApiError, jsonOrThrow } from "@ec-marketplace/api-client";

interface SellersListResponse {
  items: Seller[];
  total: number;
}

interface PlansListResponse {
  items: SubscriptionPlan[];
  total: number;
}

/**
 * Soft-fail list helper: returns an empty list on network / auth
 * errors so the admin UI renders its empty state instead of crashing
 * while admin auth is still being set up.
 */
async function softList<T extends { items: unknown[] }>(path: string, empty: T): Promise<T> {
  try {
    const res = await fetchAPI(path);
    if (!res.ok) return empty;
    return (await res.json()) as T;
  } catch {
    return empty;
  }
}

// ---------------------------------------------------------------------------
// Sellers
// ---------------------------------------------------------------------------

export async function listSellers(): Promise<Seller[]> {
  const res = await softList<SellersListResponse>("/api/v1/admin/sellers", {
    items: [],
    total: 0,
  });
  return res.items ?? [];
}

// Approves a pending seller application. Only makes sense when the
// seller is currently in pending state.
export async function approveAdminSeller(id: string): Promise<void> {
  const res = await fetchAPI(`/api/v1/admin/sellers/${encodeURIComponent(id)}/approve`, {
    method: "PUT",
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(body || `approve failed: ${res.status}`);
  }
}

// ---------------------------------------------------------------------------
// Commissions
// ---------------------------------------------------------------------------

interface CommissionsListResponse {
  items: CommissionRule[];
  total: number;
}

export async function listAdminCommissions(): Promise<CommissionRule[]> {
  const res = await softList<CommissionsListResponse>("/api/v1/admin/commissions", {
    items: [],
    total: 0,
  });
  return res.items ?? [];
}

export interface CreateCommissionInput {
  seller_id: string | null;
  category_id: string | null;
  rate_bps: number;
  priority: number;
  valid_from: string;
  valid_until: string | null;
}

export async function createAdminCommission(input: CreateCommissionInput): Promise<CommissionRule> {
  const res = await fetchAPI("/api/v1/admin/commissions", {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<CommissionRule>(res);
}

// ---------------------------------------------------------------------------
// Plans
// ---------------------------------------------------------------------------

export async function listAdminPlans(): Promise<SubscriptionPlan[]> {
  const res = await softList<PlansListResponse>("/api/v1/admin/plans", {
    items: [],
    total: 0,
  });
  return res.items ?? [];
}

export interface CreatePlanInput {
  tenant_id: string;
  name: string;
  slug: string;
  tier: number;
  price_amount: number;
  price_currency: string;
  features: {
    max_products: number;
    search_boost: number;
    featured_slots: number;
    promoted_results: number;
  };
  stripe_price_id: string;
}

export async function createAdminPlan(input: CreatePlanInput): Promise<SubscriptionPlan> {
  const res = await fetchAPI(`/api/v1/admin/plans`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<SubscriptionPlan>(res);
}

// ---------------------------------------------------------------------------
// Coupons
// ---------------------------------------------------------------------------

export async function listAdminCoupons(
  params: { status?: string; limit?: number; offset?: number } = {}
): Promise<CouponListResponse> {
  const qs = new URLSearchParams();
  if (params.status) qs.set("status", params.status);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  try {
    const res = await fetchAPI(`/api/v1/admin/coupons${qs.toString() ? `?${qs}` : ""}`);
    if (!res.ok) return { coupons: [], total: 0 };
    return (await res.json()) as CouponListResponse;
  } catch {
    return { coupons: [], total: 0 };
  }
}

export async function getAdminCoupon(id: string): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/admin/coupons/${id}`);
  return jsonOrThrow<Coupon>(res);
}

export async function createAdminCoupon(input: CreateCouponRequest): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/admin/coupons`, {
    method: "POST",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<Coupon>(res);
}

/**
 * Editable subset of a coupon. Every call must send the full payload
 * (PUT semantics) — nullable pointers cleared by sending null, set by
 * sending a value. Immutable fields (code, discount type/amount,
 * currency, valid_from) are omitted because the backend rejects
 * changes to them.
 */
export interface UpdateCouponInput {
  title: string;
  description: string;
  min_order_amount: number;
  max_discount_amount: number | null;
  expires_at_unix: number | null;
  usage_limit_total: number | null;
  usage_limit_per_user: number | null;
}

export async function updateAdminCoupon(id: string, input: UpdateCouponInput): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/admin/coupons/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
  return jsonOrThrow<Coupon>(res);
}

export async function revokeAdminCoupon(id: string): Promise<Coupon> {
  const res = await fetchAPI(`/api/v1/admin/coupons/${id}/revoke`, {
    method: "POST",
  });
  return jsonOrThrow<Coupon>(res);
}

export async function getAdminCouponStats(id: string): Promise<CouponStats> {
  const res = await fetchAPI(`/api/v1/admin/coupons/${id}/stats`);
  return jsonOrThrow<CouponStats>(res);
}

// ---------------------------------------------------------------------------
// Loyalty (admin)
// ---------------------------------------------------------------------------

export async function getAdminBuyerBalance(buyerAuth0Id: string): Promise<PointsBalance> {
  const res = await fetchAPI(
    `/api/v1/admin/loyalty/buyers/${encodeURIComponent(buyerAuth0Id)}/balance`
  );
  return jsonOrThrow<PointsBalance>(res);
}

export async function listAdminBuyerTransactions(
  buyerAuth0Id: string,
  params: { limit?: number; offset?: number } = {}
): Promise<PointsTransactionListResponse> {
  const qs = new URLSearchParams();
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const res = await fetchAPI(
    `/api/v1/admin/loyalty/buyers/${encodeURIComponent(buyerAuth0Id)}/transactions${
      qs.toString() ? `?${qs}` : ""
    }`
  );
  return jsonOrThrow<PointsTransactionListResponse>(res);
}

export async function adjustAdminBuyerPoints(
  buyerAuth0Id: string,
  input: AdjustPointsRequest
): Promise<AdjustPointsResponse> {
  const res = await fetchAPI(
    `/api/v1/admin/loyalty/buyers/${encodeURIComponent(buyerAuth0Id)}/adjust`,
    {
      method: "POST",
      body: JSON.stringify(input),
    }
  );
  return jsonOrThrow<AdjustPointsResponse>(res);
}

// ---------------------------------------------------------------------------
// Refunds (admin force-cancel)
// ---------------------------------------------------------------------------

export interface AdminRefundResponse {
  id: string;
  order_id: string;
  status: string;
  reason: string;
  seller_comment?: string | null;
  stripe_refund_id?: string | null;
  processed_at?: string | null;
}

/**
 * Force-cancel an order on admin behalf. The backend orchestrates
 * Stripe refund + per-seller transfer reversals + DB writes; any
 * pre-existing pending buyer cancellation request is folded into
 * the same approval so duplicates can't hang.
 */
export async function adminRefundOrder(
  orderId: string,
  reason: string
): Promise<AdminRefundResponse> {
  const res = await fetchAPI(`/api/v1/admin/orders/${encodeURIComponent(orderId)}/refund`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });
  return jsonOrThrow<AdminRefundResponse>(res);
}
