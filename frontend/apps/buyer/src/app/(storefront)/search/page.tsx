"use client";

import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";
import PromotedBanner from "@/components/PromotedBanner";
import type { ProductHit, Facet } from "@/lib/types";

// Mock search results - will be fetched from API in production.
const mockProducts: ProductHit[] = [
  {
    id: "p1",
    tenant_id: "t1",
    seller_id: "s1",
    name: "Premium Wireless Headphones",
    slug: "premium-wireless-headphones",
    description: "High quality wireless headphones",
    status: "active",
    price_amount: 15800,
    price_currency: "JPY",
    seller_name: "Tokyo Electronics",
    category_name: "Electronics",
    score: 2.5,
    is_promoted: false,
    plan_tier: 2,
  },
  {
    id: "p2",
    tenant_id: "t1",
    seller_id: "s2",
    name: "Organic Cotton T-Shirt",
    slug: "organic-cotton-tshirt",
    description: "Soft organic cotton t-shirt",
    status: "active",
    price_amount: 3200,
    price_currency: "JPY",
    seller_name: "Osaka Fashion",
    category_name: "Fashion",
    score: 1.5,
    is_promoted: false,
    plan_tier: 1,
  },
  {
    id: "p3",
    tenant_id: "t1",
    seller_id: "s3",
    name: "Artisan Coffee Beans",
    slug: "artisan-coffee-beans",
    description: "Freshly roasted artisan coffee beans",
    status: "active",
    price_amount: 1800,
    price_currency: "JPY",
    seller_name: "Hokkaido Foods",
    category_name: "Food",
    score: 1.0,
    is_promoted: false,
    plan_tier: 0,
  },
];

const mockPromoted: ProductHit[] = [
  {
    id: "pp1",
    tenant_id: "t1",
    seller_id: "s1",
    name: "Smart Watch Pro",
    slug: "smart-watch-pro",
    description: "Latest smart watch with health monitoring",
    status: "active",
    price_amount: 39800,
    price_currency: "JPY",
    seller_name: "Tokyo Electronics",
    category_name: "Electronics",
    score: 3.0,
    is_promoted: true,
    plan_tier: 2,
  },
];

const mockFacets: Facet[] = [
  {
    field: "category",
    values: [
      { value: "Electronics", count: 24 },
      { value: "Fashion", count: 18 },
      { value: "Food", count: 12 },
    ],
  },
  {
    field: "price_range",
    values: [
      { value: "under_1000", count: 8 },
      { value: "1000_5000", count: 15 },
      { value: "5000_10000", count: 10 },
      { value: "10000_50000", count: 7 },
    ],
  },
];

const formatPrice = (amount: number, currency: string) => {
  if (currency === "JPY") {
    return `\u00A5${amount.toLocaleString()}`;
  }
  return `${amount.toLocaleString()} ${currency}`;
};

const priceRangeLabels: Record<string, string> = {
  under_1000: "< \u00A51,000",
  "1000_5000": "\u00A51,000 - \u00A55,000",
  "5000_10000": "\u00A55,000 - \u00A510,000",
  "10000_50000": "\u00A510,000 - \u00A550,000",
  "50000_plus": "\u00A550,000+",
};

export default function SearchPage() {
  const searchParams = useSearchParams();
  const query = searchParams.get("q") || "";
  const t = useTranslations();

  // TODO: fetch from /api/v1/buyer/search?q=...
  const products = mockProducts;
  const promotedProducts = mockPromoted;
  const facets = mockFacets;
  const total = products.length;

  const categoryFacet = facets.find((f) => f.field === "category");
  const priceFacet = facets.find((f) => f.field === "price_range");

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">
          {query ? t("search.resultsFor", { query }) : t("search.title")}
        </h1>
        <p className="text-sm text-gray-500 mt-1">{t("search.totalResults", { count: total })}</p>
      </div>

      <div className="flex gap-8">
        {/* Facets sidebar */}
        <aside className="w-56 shrink-0 hidden lg:block">
          {categoryFacet && (
            <div className="mb-6">
              <h3 className="text-sm font-semibold text-gray-900 mb-3">{t("search.categories")}</h3>
              <ul className="space-y-2">
                {categoryFacet.values.map((v) => (
                  <li key={v.value}>
                    <Link
                      href={`/search?q=${encodeURIComponent(query)}&category=${encodeURIComponent(v.value)}`}
                      className="text-sm text-gray-600 hover:text-blue-600 flex justify-between"
                    >
                      <span>{v.value}</span>
                      <span className="text-gray-400">({v.count})</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {priceFacet && (
            <div className="mb-6">
              <h3 className="text-sm font-semibold text-gray-900 mb-3">{t("search.priceRange")}</h3>
              <ul className="space-y-2">
                {priceFacet.values.map((v) => (
                  <li key={v.value}>
                    <Link
                      href={`/search?q=${encodeURIComponent(query)}&price=${v.value}`}
                      className="text-sm text-gray-600 hover:text-blue-600 flex justify-between"
                    >
                      <span>{priceRangeLabels[v.value] || v.value}</span>
                      <span className="text-gray-400">({v.count})</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </aside>

        {/* Results */}
        <div className="flex-1">
          {/* Promoted products */}
          <PromotedBanner products={promotedProducts} />

          {/* Organic results */}
          {products.length === 0 ? (
            <p className="text-gray-500 text-center py-12">{t("search.noResults")}</p>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {products.map((product) => (
                <Link
                  key={product.id}
                  href={`/products/${product.slug}`}
                  className="group block rounded-lg border border-gray-200 bg-white overflow-hidden transition-shadow hover:shadow-md"
                >
                  <div className="aspect-square bg-gray-100 flex items-center justify-center">
                    <svg
                      aria-hidden="true"
                      className="w-16 h-16 text-gray-300"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={1.5}
                        d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z"
                      />
                    </svg>
                  </div>
                  <div className="p-4">
                    <h3 className="text-sm font-medium text-gray-900 group-hover:text-blue-600 transition-colors line-clamp-2">
                      {product.name}
                    </h3>
                    <p className="mt-1 text-xs text-gray-500">{product.seller_name}</p>
                    <p className="mt-2 text-lg font-bold text-gray-900">
                      {formatPrice(product.price_amount, product.price_currency)}
                    </p>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
