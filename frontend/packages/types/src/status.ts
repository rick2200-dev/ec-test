/**
 * Status enum types, kept in sync with the Go domain constants in
 * `backend/services/catalog/internal/domain` and
 * `backend/services/order/internal/domain`.
 */

/** Product status matching Go domain ProductStatus. */
export type ProductStatus = "draft" | "active" | "archived";

/** Seller status matching Go domain SellerStatus. */
export type SellerStatus = "pending" | "approved" | "rejected" | "suspended";

/** Order status matching Go domain order status constants. */
export type OrderStatus =
  | "pending"
  | "paid"
  | "processing"
  | "shipped"
  | "delivered"
  | "completed"
  | "cancelled";
