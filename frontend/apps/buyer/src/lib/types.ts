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
