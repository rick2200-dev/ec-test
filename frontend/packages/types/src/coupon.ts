/**
 * Coupon types mirror coupon-svc (`backend/services/coupon`). Discount
 * amounts are int64 minor units (yen); percent discounts are basis
 * points (1000 = 10%).
 */

export type CouponDiscountType = "percent" | "fixed_amount";
export type CouponIssuerType = "platform" | "seller";
export type CouponStatus = "active" | "revoked";

export interface Coupon {
  id: string;
  code: string;
  issuer_type: CouponIssuerType;
  issuer_id?: string | null;
  discount_type: CouponDiscountType;
  discount_percent_bps?: number | null;
  discount_amount?: number | null;
  currency: string;
  min_order_amount: number;
  max_discount_amount?: number | null;
  valid_from: string;
  expires_at?: string | null;
  usage_limit_total?: number | null;
  usage_limit_per_user?: number | null;
  usage_count: number;
  status: CouponStatus;
  title: string;
  description: string;
  created_at: string;
  updated_at: string;
}

/** Per-seller subtotal sent in the preview / reserve request. */
export interface SellerSubtotal {
  seller_id: string;
  subtotal: number;
}

export interface CouponPreviewRequest {
  code: string;
  currency: string;
  seller_subtotals: SellerSubtotal[];
}

/**
 * Coupon preview response.
 *
 * NOTE: backend `port.PreviewResult` is currently emitted by
 * `httputil.JSON` without struct json tags — Go falls back to PascalCase
 * (`CouponID`, `Title`, ...). PR 3 will add a snake_case response
 * envelope on the backend before wiring this; downstream callers should
 * treat this as the *target* shape.
 */
export interface CouponPreviewResult {
  coupon_id: string;
  title: string;
  description: string;
  discount_amount: number;
  applicable_seller_id?: string | null;
  expires_at?: string | null;
}

export interface CouponRedemption {
  id: string;
  coupon_id: string;
  buyer_auth0_id: string;
  order_id: string;
  discount_applied: number;
  refunded_amount: number;
  reservation_id: string;
  committed_at: string;
  refunded_at?: string | null;
  refunded_reason?: string;
}

export interface CouponRedemptionListResponse {
  redemptions: CouponRedemption[];
  total: number;
}

export interface CouponListResponse {
  coupons: Coupon[];
  total: number;
}

export interface CouponStats {
  coupon_id: string;
  redeemed_count: number;
  total_discount_applied: number;
  pending_reservation_count: number;
}

export interface CreateCouponRequest {
  code: string;
  issuer_type?: CouponIssuerType;
  issuer_id?: string;
  discount_type: CouponDiscountType;
  discount_percent_bps?: number;
  discount_amount?: number;
  currency: string;
  min_order_amount: number;
  max_discount_amount?: number;
  valid_from_unix?: number;
  expires_at_unix?: number;
  usage_limit_total?: number;
  usage_limit_per_user?: number;
  title: string;
  description: string;
}
