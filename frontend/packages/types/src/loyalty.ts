/**
 * Loyalty (points) types mirror loyalty-svc
 * (`backend/services/loyalty`). Amounts are int64 minor units; ledger
 * `amount` is signed (positive earn, negative redeem).
 */

export interface PointsBalance {
  buyer_auth0_id: string;
  balance: number;
  pending_redemption: number;
  spendable: number;
  lifetime_earned: number;
  lifetime_redeemed: number;
  updated_at: string;
}

export type PointsTransactionType =
  | "earn"
  | "redeem"
  | "refund"
  | "reverse_earn"
  | "adjust"
  | "expire";

export type PointsSourceType =
  | "order_paid"
  | "reservation_commit"
  | "order_cancelled"
  | "admin_adjust"
  | "expiration_job";

export interface PointsTransaction {
  id: string;
  buyer_auth0_id: string;
  type: PointsTransactionType;
  amount: number;
  balance_after: number;
  source_type: PointsSourceType;
  source_id: string;
  reservation_id?: string | null;
  expires_at?: string | null;
  note?: string;
  created_at: string;
}

export interface PointsTransactionListResponse {
  transactions: PointsTransaction[];
  total: number;
}

/** Admin-only manual adjustment payload. `amount` is signed. */
export interface AdjustPointsRequest {
  amount: number;
  reason: string;
  admin_user_id?: string;
}

export interface AdjustPointsResponse {
  transaction: PointsTransaction;
  new_balance: number;
}
