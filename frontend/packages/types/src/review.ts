/**
 * Review types mirror the shapes returned by the review service
 * (`backend/services/review`). Both buyer and seller apps consume the
 * same JSON, so the types live here rather than being duplicated.
 */

/** A buyer's review of a product. */
export interface Review {
  id: string;
  tenant_id: string;
  buyer_auth0_id: string;
  product_id: string;
  seller_id: string;
  product_name: string;
  rating: number;
  title: string;
  body: string;
  created_at: string;
  updated_at: string;
  reply?: ReviewReply | null;
}

/** A seller's reply to a review. */
export interface ReviewReply {
  id: string;
  tenant_id: string;
  review_id: string;
  seller_auth0_id: string;
  body: string;
  created_at: string;
  updated_at: string;
}

/** Aggregate rating for a product. */
export interface ProductRating {
  tenant_id: string;
  product_id: string;
  average_rating: number;
  review_count: number;
  updated_at: string;
}

/** Paginated review list response. */
export interface ReviewListResponse {
  items: Review[];
  total: number;
  limit: number;
  offset: number;
}
