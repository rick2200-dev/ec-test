"use client";

import { useEffect, useState } from "react";
import { fetchAPI } from "@/lib/api";
import { Product } from "@/lib/types";
import { products as mockProducts } from "@/lib/mock-data";
import ProductCard from "@/components/ProductCard";

interface RecommendationsProps {
  type: "popular" | "similar" | "for_you";
  productId?: string;
  title: string;
}

export function Recommendations({ type, productId, title }: RecommendationsProps) {
  const [items, setItems] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const params = new URLSearchParams({ type });
        if (productId) params.set("product_id", productId);

        const res = await fetchAPI(`/api/v1/buyer/recommendations?${params.toString()}`);
        if (!cancelled && res.ok) {
          const data = await res.json();
          if (data.products && data.products.length > 0) {
            setItems(data.products);
            setLoading(false);
            return;
          }
        }
      } catch {
        // Fall through to mock data
      }

      // Fallback: use mock data
      if (!cancelled) {
        const fallback = mockProducts.filter((p) => p.status === "active").slice(0, 6);
        setItems(fallback);
        setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [type, productId]);

  if (loading) {
    return (
      <section
        aria-live="polite"
        aria-busy={true}
        className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8"
      >
        <h2 className="text-2xl font-bold text-gray-900">{title}</h2>
        <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              aria-hidden="true"
              className="animate-pulse rounded-lg border border-gray-200 bg-white overflow-hidden"
            >
              <div className="aspect-square bg-gray-200" />
              <div className="p-4 space-y-2">
                <div className="h-4 bg-gray-200 rounded w-3/4" />
                <div className="h-4 bg-gray-200 rounded w-1/2" />
              </div>
            </div>
          ))}
        </div>
      </section>
    );
  }

  if (items.length === 0) return null;

  return (
    <section
      aria-live="polite"
      aria-busy={false}
      className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8"
    >
      <h2 className="text-2xl font-bold text-gray-900">{title}</h2>
      <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
        {items.map((product) => (
          <ProductCard key={product.id} product={product} />
        ))}
      </div>
    </section>
  );
}
