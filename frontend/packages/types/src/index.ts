/**
 * Shared domain types used by multiple frontend apps.
 *
 * Only add types here when they are used by at least two of buyer/seller/admin
 * and have an identical shape. App-specific types should stay in each app's
 * `src/lib/types.ts`.
 */

export type { Inquiry, InquiryMessage, InquiryWithMessages, InquiryListResponse } from "./inquiry";

export type { PlanFeatures } from "./subscription";

export type { ProductStatus, SellerStatus, OrderStatus } from "./status";

export type {
  CancellationRequest,
  CancellationRequestStatus,
  CancellationRequestListResponse,
} from "./cancellation";

export type { Review, ReviewReply, ProductRating, ReviewListResponse } from "./review";

export type { Cart, CartItem, ShippingAddress, CheckoutRequest, CheckoutResult } from "./cart";

export type {
  Coupon,
  CouponDiscountType,
  CouponIssuerType,
  CouponStatus,
  SellerSubtotal,
  CouponPreviewRequest,
  CouponPreviewResult,
  CouponRedemption,
  CouponRedemptionListResponse,
  CouponListResponse,
  CouponStats,
  CreateCouponRequest,
} from "./coupon";

export type {
  PointsBalance,
  PointsTransaction,
  PointsTransactionType,
  PointsSourceType,
  PointsTransactionListResponse,
  AdjustPointsRequest,
  AdjustPointsResponse,
} from "./loyalty";

export type {
  Shipment,
  ShipmentStatus,
  ShipmentListResponse,
  RegisterShipmentRequest,
  MarkDeliveredRequest,
} from "./shipment";

export type {
  ProductSearchHit,
  SearchFacet,
  SearchFacetValue,
  SearchResponse,
  SearchRequestParams,
  SearchSuggestion,
  SearchSuggestResponse,
} from "./search";
