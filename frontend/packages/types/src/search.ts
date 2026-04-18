/**
 * Search types mirror search-svc (`backend/services/search`). Prices
 * arrive as `float64` from the search index — they are the canonical
 * display amounts (not int64 minor units), unlike the rest of the
 * system. Treat them as numbers and format on display.
 */

export interface ProductSearchHit {
  id: string;
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

export interface SearchFacetValue {
  value: string;
  count: number;
}

export interface SearchFacet {
  field: string;
  values: SearchFacetValue[];
}

export interface SearchResponse {
  products: ProductSearchHit[];
  promoted_products?: ProductSearchHit[];
  total: number;
  facets: SearchFacet[];
}

export interface SearchRequestParams {
  q?: string;
  seller_id?: string;
  category_id?: string;
  min_price?: number;
  max_price?: number;
  status?: string;
  sort_by?: string;
  sort_order?: "asc" | "desc";
  limit?: number;
  offset?: number;
}

export interface SearchSuggestion {
  text: string;
}

export interface SearchSuggestResponse {
  suggestions: SearchSuggestion[];
}
