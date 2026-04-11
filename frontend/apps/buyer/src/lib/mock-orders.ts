import { products, skus, getSellerById } from "./mock-data";

export type BuyerOrderStatus =
  | "pending"
  | "paid"
  | "processing"
  | "shipped"
  | "delivered"
  | "completed"
  | "cancelled";

export interface BuyerOrderLine {
  sku_id: string;
  product_id: string;
  product_name: string;
  sku_code: string;
  quantity: number;
  unit_price: number;
  seller_id: string;
  seller_name: string;
}

export interface BuyerOrder {
  id: string;
  status: BuyerOrderStatus;
  total_amount: number;
  currency: string;
  ordered_at: string;
  lines: BuyerOrderLine[];
}

/**
 * Mock order history for the buyer. The real implementation will fetch
 * from /api/v1/buyer/orders once that endpoint is built. Keeping this
 * local lets the inquiry UI work end-to-end against the demo seed data.
 */
function buildLine(skuId: string, quantity: number): BuyerOrderLine {
  const sku = skus.find((s) => s.id === skuId);
  if (!sku) {
    throw new Error(`mock-orders: unknown sku ${skuId}`);
  }
  const product = products.find((p) => p.id === sku.product_id);
  const seller = getSellerById(sku.seller_id);
  return {
    sku_id: sku.id,
    product_id: sku.product_id,
    product_name: product?.name ?? "Unknown product",
    sku_code: sku.sku_code,
    quantity,
    unit_price: sku.price_amount,
    seller_id: sku.seller_id,
    seller_name: seller?.name ?? "Unknown seller",
  };
}

export const buyerOrders: BuyerOrder[] = [
  {
    id: "ord_00000000000000000000000001",
    status: "shipped",
    total_amount: 12800,
    currency: "JPY",
    ordered_at: "2026-04-02T09:12:00Z",
    lines: [buildLine("44eebc99-9c0b-4ef8-bb6d-6bb9bd380c01", 1)],
  },
  {
    id: "ord_00000000000000000000000002",
    status: "paid",
    total_amount: 7000,
    currency: "JPY",
    ordered_at: "2026-04-05T18:44:00Z",
    lines: [buildLine("66eebc99-9c0b-4ef8-bb6d-6bb9bd380c03", 2)],
  },
  {
    id: "ord_00000000000000000000000003",
    status: "pending",
    total_amount: 5980,
    currency: "JPY",
    ordered_at: "2026-04-09T12:01:00Z",
    lines: [buildLine("88eebc99-9c0b-4ef8-bb6d-6bb9bd380c05", 1)],
  },
];

export function getBuyerOrderById(id: string): BuyerOrder | undefined {
  return buyerOrders.find((o) => o.id === id);
}

/** Order status at which buyer → seller inquiries are allowed. */
export function canContactSeller(status: BuyerOrderStatus): boolean {
  return (
    status === "paid" ||
    status === "processing" ||
    status === "shipped" ||
    status === "delivered" ||
    status === "completed"
  );
}
