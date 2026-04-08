"use client";

import { useState } from "react";
import Link from "next/link";
import { getProductWithSKUs, formatPrice } from "@/lib/mock-data";
import { notFound } from "next/navigation";
import { use } from "react";
import { ProductViewTracker } from "@/components/ProductViewTracker";
import { useTranslations } from "next-intl";

interface ProductDetailPageProps {
  params: Promise<{ slug: string }>;
}

export default function ProductDetailPage({ params }: ProductDetailPageProps) {
  const { slug } = use(params);
  const data = getProductWithSKUs(slug);

  if (!data) {
    notFound();
  }

  const { product, skus, seller, category } = data;
  const t = useTranslations();

  // Extract unique attribute keys from SKUs
  const attrKeys = new Set<string>();
  skus.forEach((sku) => {
    if (sku.attributes) {
      Object.keys(sku.attributes).forEach((k) => attrKeys.add(k));
    }
  });

  // Build attribute options
  const attrOptions: Record<string, string[]> = {};
  attrKeys.forEach((key) => {
    const values = new Set<string>();
    skus.forEach((sku) => {
      const val = sku.attributes?.[key];
      if (val && typeof val === "string") values.add(val);
    });
    attrOptions[key] = Array.from(values);
  });

  const [selectedAttrs, setSelectedAttrs] = useState<Record<string, string>>(() => {
    const defaults: Record<string, string> = {};
    Object.entries(attrOptions).forEach(([key, values]) => {
      if (values.length > 0) defaults[key] = values[0];
    });
    return defaults;
  });

  // Find matching SKU
  const selectedSku = skus.find((sku) => {
    if (!sku.attributes) return Object.keys(selectedAttrs).length === 0;
    return Object.entries(selectedAttrs).every(([key, val]) => sku.attributes?.[key] === val);
  });

  const attrLabels: Record<string, string> = {
    color: t("attr.color"),
    size: t("attr.size"),
    material: t("attr.material"),
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <ProductViewTracker productId={product.id} />
      {/* Breadcrumb */}
      <nav aria-label={t("a11y.breadcrumb")} className="mb-6 text-sm text-gray-500">
        <Link href="/" className="hover:text-gray-700">
          {t("product.breadcrumbTop")}
        </Link>
        <span className="mx-2">/</span>
        <Link href="/products" className="hover:text-gray-700">
          {t("product.productList")}
        </Link>
        {category && (
          <>
            <span className="mx-2">/</span>
            <Link href={`/products?category=${category.slug}`} className="hover:text-gray-700">
              {category.name}
            </Link>
          </>
        )}
        <span className="mx-2">/</span>
        <span className="text-gray-900">{product.name}</span>
      </nav>

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-2">
        {/* Product image placeholder */}
        <div className="aspect-square rounded-xl bg-gray-100 flex items-center justify-center">
          <svg
            aria-hidden="true"
            className="w-32 h-32 text-gray-300"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z"
            />
          </svg>
        </div>

        {/* Product info */}
        <div>
          {category && (
            <Link
              href={`/products?category=${category.slug}`}
              className="inline-block text-xs font-medium text-blue-600 hover:text-blue-800"
            >
              {category.name}
            </Link>
          )}
          <h1 className="mt-1 text-3xl font-bold text-gray-900">{product.name}</h1>

          {/* Price */}
          <p className="mt-4 text-3xl font-bold text-gray-900">
            {selectedSku
              ? formatPrice(selectedSku.price_amount, selectedSku.price_currency)
              : formatPrice(skus[0]?.price_amount ?? 0)}
          </p>

          {selectedSku && <p className="mt-1 text-xs text-gray-500">SKU: {selectedSku.sku_code}</p>}

          {/* Description */}
          <p className="mt-6 text-gray-600 leading-relaxed">{product.description}</p>

          {/* Variant selectors */}
          {Object.entries(attrOptions).map(([key, values]) => (
            <div key={key} className="mt-6">
              <label className="block text-sm font-medium text-gray-900">
                {attrLabels[key] ?? key}
              </label>
              <div className="mt-2 flex flex-wrap gap-2">
                {values.map((val) => (
                  <button
                    key={val}
                    onClick={() => setSelectedAttrs((prev) => ({ ...prev, [key]: val }))}
                    aria-pressed={selectedAttrs[key] === val}
                    className={`rounded-md border px-4 py-2 text-sm transition-colors ${
                      selectedAttrs[key] === val
                        ? "border-blue-600 bg-blue-50 text-blue-600 font-medium"
                        : "border-gray-300 text-gray-700 hover:border-gray-400"
                    }`}
                  >
                    {val}
                  </button>
                ))}
              </div>
            </div>
          ))}

          {/* Add to cart */}
          <button className="mt-8 w-full rounded-lg bg-blue-600 px-8 py-3 text-sm font-semibold text-white shadow-sm hover:bg-blue-700 transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2">
            {t("product.addToCart")}
          </button>

          {/* Seller info */}
          {seller && (
            <div className="mt-8 rounded-lg border border-gray-200 p-4">
              <h2 className="text-sm font-semibold text-gray-900">{t("product.sellerInfo")}</h2>
              <div className="mt-3 flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-100 text-sm font-bold text-gray-600">
                  {seller.name.charAt(0)}
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">{seller.name}</p>
                  <p className="text-xs text-gray-500">
                    {seller.status === "approved"
                      ? t("product.verifiedSeller")
                      : t("product.seller")}
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
