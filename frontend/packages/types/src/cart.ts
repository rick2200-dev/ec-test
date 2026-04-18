/**
 * Cart and checkout types mirror the JSON shapes returned by cart-svc
 * (`backend/services/cart`). Amounts are int64 minor units (yen) and
 * arrive as JS numbers — JPY fits well within Number.MAX_SAFE_INTEGER
 * for any realistic order so no BigInt handling is needed yet.
 */

export interface CartItem {
  sku_id: string;
  seller_id: string;
  quantity: number;
  unit_price_snapshot: number;
  currency: string;
  product_name_snapshot: string;
  sku_code_snapshot: string;
  added_at: string;
}

export interface Cart {
  buyer_auth0_id: string;
  items: CartItem[];
  updated_at: string;
}

/**
 * Shipping address is forwarded verbatim by cart-svc to order-svc as
 * `json.RawMessage`. We use a recursive object so apps can attach any
 * structured address shape (the validation lives downstream).
 */
export type ShippingAddress = Record<string, unknown>;

export interface CheckoutRequest {
  shipping_address: ShippingAddress;
  currency: string;
  /** Optional coupon code; omit or empty string to skip. */
  coupon_code?: string;
  /** Optional points to redeem in minor units; omit or 0 to skip. */
  points_to_redeem?: number;
}

export interface CheckoutResult {
  order_ids: string[];
  stripe_client_secret: string;
  stripe_payment_intent_id: string;
  total_amount: number;
  currency: string;
}
