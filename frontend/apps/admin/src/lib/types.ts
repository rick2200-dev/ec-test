// Mirrors backend auth-svc domain.Seller (JSON shape). `status` also
// allows "rejected" which the UI renders as "suspended" for now (the
// tabs treat non-active sellers uniformly).
export interface Seller {
  id: string;
  auth0_org_id: string;
  name: string;
  slug: string;
  status: "pending" | "approved" | "rejected" | "suspended";
  stripe_account_id: string;
  commission_rate_bps: number;
  created_at: string;
  updated_at: string;
}

// Mirrors backend order-svc domain.CommissionRule.
export interface CommissionRule {
  id: string;
  seller_id: string | null;
  category_id: string | null;
  rate_bps: number;
  priority: number;
  valid_from: string;
  valid_until: string | null;
  created_at: string;
}

export interface PlatformStats {
  totalSellers: number;
  monthlyTransactionAmount: number;
  monthlyCommissionIncome: number;
}

import type { PlanFeatures } from "@ec-marketplace/types";
export type { PlanFeatures };

export interface SubscriptionPlan {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  tier: number;
  price_amount: number;
  price_currency: string;
  features: PlanFeatures;
  stripe_price_id: string;
  status: "active" | "archived";
  created_at: string;
  updated_at: string;
}
