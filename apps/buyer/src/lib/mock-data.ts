import { Category, Product, SKU, Seller, ProductWithSKUs } from "./types";

const TENANT_ID = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11";

export const sellers: Seller[] = [
  {
    id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    tenant_id: TENANT_ID,
    name: "Tokyo Electronics",
    slug: "tokyo-electronics",
    status: "approved",
  },
  {
    id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    tenant_id: TENANT_ID,
    name: "Osaka Fashion",
    slug: "osaka-fashion",
    status: "approved",
  },
  {
    id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
    tenant_id: TENANT_ID,
    name: "Kyoto Crafts",
    slug: "kyoto-crafts",
    status: "pending",
  },
];

export const categories: Category[] = [
  {
    id: "e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55",
    tenant_id: TENANT_ID,
    name: "Electronics",
    slug: "electronics",
    sort_order: 1,
    created_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66",
    tenant_id: TENANT_ID,
    name: "Fashion",
    slug: "fashion",
    sort_order: 2,
    created_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "06eebc99-9c0b-4ef8-bb6d-6bb9bd380a77",
    tenant_id: TENANT_ID,
    name: "Handmade",
    slug: "handmade",
    sort_order: 3,
    created_at: "2025-01-01T00:00:00Z",
  },
];

export const products: Product[] = [
  {
    id: "11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01",
    tenant_id: TENANT_ID,
    seller_id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    category_id: "e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55",
    name: "Wireless Headphones",
    slug: "wireless-headphones",
    description: "High-quality wireless headphones with noise cancellation",
    status: "active",
    attributes: {},
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02",
    tenant_id: TENANT_ID,
    seller_id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    category_id: "f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66",
    name: "Cotton T-Shirt",
    slug: "cotton-tshirt",
    description: "Premium organic cotton t-shirt",
    status: "active",
    attributes: {},
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "33eebc99-9c0b-4ef8-bb6d-6bb9bd380b03",
    tenant_id: TENANT_ID,
    seller_id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    category_id: "e4eebc99-9c0b-4ef8-bb6d-6bb9bd380a55",
    name: "USB-C Hub",
    slug: "usb-c-hub",
    description: "7-in-1 USB-C hub with 4K HDMI",
    status: "active",
    attributes: {},
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "44eebc99-9c0b-4ef8-bb6d-6bb9bd380b04",
    tenant_id: TENANT_ID,
    seller_id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
    category_id: "06eebc99-9c0b-4ef8-bb6d-6bb9bd380a77",
    name: "Ceramic Tea Cup",
    slug: "ceramic-tea-cup",
    description: "Handcrafted ceramic tea cup with traditional Japanese design",
    status: "active",
    attributes: {},
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "55eebc99-9c0b-4ef8-bb6d-6bb9bd380b05",
    tenant_id: TENANT_ID,
    seller_id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    category_id: "f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66",
    name: "Denim Jacket",
    slug: "denim-jacket",
    description: "Classic denim jacket with modern fit",
    status: "active",
    attributes: {},
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "66eebc99-9c0b-4ef8-bb6d-6bb9bd380b06",
    tenant_id: TENANT_ID,
    seller_id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
    category_id: "06eebc99-9c0b-4ef8-bb6d-6bb9bd380a77",
    name: "Wooden Chopsticks Set",
    slug: "wooden-chopsticks-set",
    description: "Hand-carved wooden chopsticks set with lacquer finish",
    status: "active",
    attributes: {},
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
];

export const skus: SKU[] = [
  {
    id: "44eebc99-9c0b-4ef8-bb6d-6bb9bd380c01",
    tenant_id: TENANT_ID,
    product_id: "11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01",
    seller_id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    sku_code: "WH-BLK-001",
    price_amount: 12800,
    price_currency: "JPY",
    attributes: { color: "black" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "55eebc99-9c0b-4ef8-bb6d-6bb9bd380c02",
    tenant_id: TENANT_ID,
    product_id: "11eebc99-9c0b-4ef8-bb6d-6bb9bd380b01",
    seller_id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    sku_code: "WH-WHT-001",
    price_amount: 12800,
    price_currency: "JPY",
    attributes: { color: "white" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "66eebc99-9c0b-4ef8-bb6d-6bb9bd380c03",
    tenant_id: TENANT_ID,
    product_id: "22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02",
    seller_id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    sku_code: "TS-M-BLK",
    price_amount: 3500,
    price_currency: "JPY",
    attributes: { size: "M", color: "black" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "77eebc99-9c0b-4ef8-bb6d-6bb9bd380c04",
    tenant_id: TENANT_ID,
    product_id: "22eebc99-9c0b-4ef8-bb6d-6bb9bd380b02",
    seller_id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    sku_code: "TS-L-BLK",
    price_amount: 3500,
    price_currency: "JPY",
    attributes: { size: "L", color: "black" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "88eebc99-9c0b-4ef8-bb6d-6bb9bd380c05",
    tenant_id: TENANT_ID,
    product_id: "33eebc99-9c0b-4ef8-bb6d-6bb9bd380b03",
    seller_id: "b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22",
    sku_code: "HUB-7IN1",
    price_amount: 5980,
    price_currency: "JPY",
    attributes: {},
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "99eebc99-9c0b-4ef8-bb6d-6bb9bd380c06",
    tenant_id: TENANT_ID,
    product_id: "44eebc99-9c0b-4ef8-bb6d-6bb9bd380b04",
    seller_id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
    sku_code: "CUP-BLU-001",
    price_amount: 4200,
    price_currency: "JPY",
    attributes: { color: "blue" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "aaeebc99-9c0b-4ef8-bb6d-6bb9bd380c07",
    tenant_id: TENANT_ID,
    product_id: "44eebc99-9c0b-4ef8-bb6d-6bb9bd380b04",
    seller_id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
    sku_code: "CUP-GRN-001",
    price_amount: 4200,
    price_currency: "JPY",
    attributes: { color: "green" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "bbeebc99-9c0b-4ef8-bb6d-6bb9bd380c08",
    tenant_id: TENANT_ID,
    product_id: "55eebc99-9c0b-4ef8-bb6d-6bb9bd380b05",
    seller_id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    sku_code: "DJ-M-IND",
    price_amount: 9800,
    price_currency: "JPY",
    attributes: { size: "M", color: "indigo" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "cceebc99-9c0b-4ef8-bb6d-6bb9bd380c09",
    tenant_id: TENANT_ID,
    product_id: "55eebc99-9c0b-4ef8-bb6d-6bb9bd380b05",
    seller_id: "c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33",
    sku_code: "DJ-L-IND",
    price_amount: 9800,
    price_currency: "JPY",
    attributes: { size: "L", color: "indigo" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
  {
    id: "ddeebc99-9c0b-4ef8-bb6d-6bb9bd380c10",
    tenant_id: TENANT_ID,
    product_id: "66eebc99-9c0b-4ef8-bb6d-6bb9bd380b06",
    seller_id: "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
    sku_code: "CHOP-NAT-001",
    price_amount: 2800,
    price_currency: "JPY",
    attributes: { material: "natural wood" },
    status: "active",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  },
];

/** Helper: get the lowest price SKU for a product */
export function getLowestPrice(productId: string): number {
  const productSkus = skus.filter((s) => s.product_id === productId);
  if (productSkus.length === 0) return 0;
  return Math.min(...productSkus.map((s) => s.price_amount));
}

/** Helper: get seller by ID */
export function getSellerById(sellerId: string): Seller | undefined {
  return sellers.find((s) => s.id === sellerId);
}

/** Helper: get category by ID */
export function getCategoryById(categoryId: string): Category | undefined {
  return categories.find((c) => c.id === categoryId);
}

/** Helper: get product with all related data */
export function getProductWithSKUs(slug: string): ProductWithSKUs | undefined {
  const product = products.find((p) => p.slug === slug);
  if (!product) return undefined;
  return {
    product,
    skus: skus.filter((s) => s.product_id === product.id),
    seller: getSellerById(product.seller_id),
    category: product.category_id ? getCategoryById(product.category_id) : undefined,
  };
}

/** Helper: format price in JPY */
export function formatPrice(amount: number, currency: string = "JPY"): string {
  return new Intl.NumberFormat("ja-JP", {
    style: "currency",
    currency,
    minimumFractionDigits: 0,
  }).format(amount);
}
