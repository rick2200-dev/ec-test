"use client";

import { useEffect, useState } from "react";
import { fetchAPI } from "@/lib/api";
import { Product } from "@/lib/types";
import {
  products as mockProducts,
  formatPrice,
  getLowestPrice,
  getSellerById,
} from "@/lib/mock-data";
import { RecommendationsPresenter, type RecommendationsItem } from "./Recommendations.presenter";

interface RecommendationsProps {
  type: "popular" | "similar" | "for_you";
  productId?: string;
  title: string;
}

function toItems(products: Product[]): RecommendationsItem[] {
  return products.map((p) => {
    const seller = getSellerById(p.seller_id);
    return {
      id: p.id,
      href: `/products/${p.slug}`,
      name: p.name,
      sellerName: seller?.name,
      priceLabel: formatPrice(getLowestPrice(p.id)),
    };
  });
}

export function Recommendations({ type, productId, title }: RecommendationsProps) {
  const [items, setItems] = useState<RecommendationsItem[]>([]);
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
            setItems(toItems(data.products));
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
        setItems(toItems(fallback));
        setLoading(false);
      }
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [type, productId]);

  return <RecommendationsPresenter title={title} loading={loading} items={items} />;
}
