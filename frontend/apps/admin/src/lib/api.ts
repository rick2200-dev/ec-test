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
import type { Seller, SubscriptionPlan } from "./types";

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
