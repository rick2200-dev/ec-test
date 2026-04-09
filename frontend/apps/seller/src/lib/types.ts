export interface Category {
  id: string;
  name: string;
  slug: string;
  parentId: string | null;
}

export interface SKU {
  id: string;
  productId: string;
  code: string;
  price: number;
  attributes: Record<string, string>;
  stockQuantity: number;
  reservedQuantity: number;
}

export interface Product {
  id: string;
  sellerId: string;
  categoryId: string;
  name: string;
  slug: string;
  description: string;
  status: "draft" | "active" | "archived";
  skus: SKU[];
  createdAt: string;
  updatedAt: string;
}

export interface Order {
  id: string;
  buyerName: string;
  items: OrderItem[];
  totalAmount: number;
  status: "pending" | "processing" | "shipped" | "completed" | "cancelled";
  createdAt: string;
}

export interface OrderItem {
  productName: string;
  skuCode: string;
  quantity: number;
  unitPrice: number;
}

export interface InventoryItem {
  skuId: string;
  skuCode: string;
  productName: string;
  stockQuantity: number;
  reservedQuantity: number;
  availableQuantity: number;
  lowStockThreshold: number;
}

export interface SalesStats {
  todaySales: number;
  monthlySales: number;
  pendingOrders: number;
  stockAlerts: number;
}

export interface PlanFeatures {
  max_products: number;
  search_boost: number;
  featured_slots: number;
  promoted_results: number;
}

export interface SubscriptionPlan {
  id: string;
  name: string;
  slug: string;
  tier: number;
  price_amount: number;
  price_currency: string;
  features: PlanFeatures;
}

export interface SellerSubscription {
  id: string;
  plan_id: string;
  plan_name: string;
  plan_slug: string;
  plan_tier: number;
  status: "active" | "past_due" | "canceled" | "trialing";
  current_period_end: string | null;
}
