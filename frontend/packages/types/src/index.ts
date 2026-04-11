/**
 * Shared domain types used by multiple frontend apps.
 *
 * Only add types here when they are used by at least two of buyer/seller/admin
 * and have an identical shape. App-specific types should stay in each app's
 * `src/lib/types.ts`.
 */

export type {
  Inquiry,
  InquiryMessage,
  InquiryWithMessages,
  InquiryListResponse,
} from "./inquiry";

export type { PlanFeatures } from "./subscription";

export type { ProductStatus, SellerStatus, OrderStatus } from "./status";
