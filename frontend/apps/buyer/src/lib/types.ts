/** Product status matching Go domain ProductStatus */
export type ProductStatus = "draft" | "active" | "archived";

/** Seller status matching Go domain SellerStatus */
export type SellerStatus = "pending" | "approved" | "rejected" | "suspended";

/** Category matching catalog_svc.Category */
export interface Category {
  id: string;
  tenant_id: string;
  parent_id?: string;
  name: string;
  slug: string;
  sort_order: number;
  created_at: string;
}

/** Product matching catalog_svc.Product */
export interface Product {
  id: string;
  tenant_id: string;
  seller_id: string;
  category_id?: string;
  name: string;
  slug: string;
  description: string;
  status: ProductStatus;
  attributes?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

/** SKU matching catalog_svc.SKU */
export interface SKU {
  id: string;
  tenant_id: string;
  product_id: string;
  seller_id: string;
  sku_code: string;
  price_amount: number;
  price_currency: string;
  attributes?: Record<string, unknown>;
  status: ProductStatus;
  created_at: string;
  updated_at: string;
}

/** Seller matching auth_svc.Seller (buyer-facing subset) */
export interface Seller {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  status: SellerStatus;
}

/** Product with associated SKUs and seller info for display */
export interface ProductWithSKUs {
  product: Product;
  skus: SKU[];
  seller?: Seller;
  category?: Category;
}

/** A single product hit returned by the search service */
export interface ProductHit {
  id: string;
  tenant_id: string;
  seller_id: string;
  name: string;
  slug: string;
  description: string;
  status: string;
  price_amount: number;
  price_currency: string;
  seller_name: string;
  category_name: string;
  score: number;
  is_promoted: boolean;
  plan_tier: number;
}

/** Facet value in search results */
export interface FacetValue {
  value: string;
  count: number;
}

/** Facet group in search results */
export interface Facet {
  field: string;
  values: FacetValue[];
}

/** Search result response from the search service */
export interface SearchResult {
  products: ProductHit[];
  promoted_products: ProductHit[];
  total: number;
  facets: Facet[];
}

/** Order status matching Go domain order status constants */
export type OrderStatus =
  | "pending"
  | "paid"
  | "processing"
  | "shipped"
  | "delivered"
  | "completed"
  | "cancelled";

/**
 * OrderSummary matches the shape returned by GET /api/v1/buyer/orders.
 * Corresponds to domain.Order on the backend — a flat listing row with no
 * line items, used for the order history index page.
 */
export interface OrderSummary {
  id: string;
  tenant_id: string;
  seller_id: string;
  /** Company name snapshot captured at checkout. Empty if seller was
   *  deleted before the snapshot existed — the UI falls back to a label. */
  seller_name: string;
  status: OrderStatus;
  subtotal_amount: number;
  shipping_fee: number;
  commission_amount: number;
  total_amount: number;
  currency: string;
  stripe_payment_intent_id?: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
}

/** Paginated response wrapper matching pagination.Response[T] on the Go side. */
export interface OrderListResponse {
  items: OrderSummary[];
  total: number;
  limit: number;
  offset: number;
}

/**
 * OrderLine is one enriched line item returned by GET /api/v1/buyer/orders/{id}.
 * product_name is always the historical snapshot; image_url / product_slug
 * reflect current catalog state, and is_deleted is true when the product
 * has been archived or removed since the purchase.
 */
export interface OrderLine {
  id: string;
  sku_id: string;
  product_id: string;
  product_name: string;
  sku_code: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  image_url: string;
  product_slug: string;
  is_deleted: boolean;
}

/** OrderDetail matches the response shape of GET /api/v1/buyer/orders/{id}. */
export interface OrderDetail {
  id: string;
  tenant_id: string;
  seller_id: string;
  seller_name: string;
  status: OrderStatus;
  subtotal_amount: number;
  shipping_fee: number;
  commission_amount: number;
  total_amount: number;
  currency: string;
  shipping_address?: Record<string, unknown>;
  stripe_payment_intent_id?: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
  lines: OrderLine[];
}

/** Inquiry thread (1 thread per buyer × seller × SKU) */
export interface Inquiry {
  id: string;
  tenant_id: string;
  buyer_auth0_id: string;
  seller_id: string;
  sku_id: string;
  product_name: string;
  sku_code: string;
  subject: string;
  status: "open" | "closed";
  last_message_at: string;
  created_at: string;
  updated_at: string;
  unread_count?: number;
}

/** Single message in an inquiry thread */
export interface InquiryMessage {
  id: string;
  tenant_id: string;
  inquiry_id: string;
  sender_type: "buyer" | "seller";
  sender_id: string;
  body: string;
  read_at?: string | null;
  created_at: string;
}

/** Inquiry thread with its messages */
export interface InquiryWithMessages extends Inquiry {
  messages: InquiryMessage[];
}

/** Paginated inquiry list response */
export interface InquiryListResponse {
  items: Inquiry[];
  total: number;
  limit: number;
  offset: number;
}
