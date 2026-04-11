export interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: "active" | "suspended" | "pending";
  sellerCount: number;
  createdAt: string;
}

export interface Seller {
  id: string;
  name: string;
  tenantName: string;
  status: "pending" | "approved" | "suspended";
  commissionRate: number;
  stripeConnected: boolean;
  createdAt: string;
}

export interface CommissionRule {
  id: string;
  tenantName: string;
  sellerName: string | null;
  category: string | null;
  rate: number;
  priority: number;
  validFrom: string;
  validUntil: string | null;
}

export interface PlatformStats {
  totalTenants: number;
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
