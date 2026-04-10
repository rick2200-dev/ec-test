"use client";

import { useEffect } from "react";
import { trackEvent } from "@/lib/api";

export function ProductViewTracker({ productId }: { productId: string }) {
  useEffect(() => {
    trackEvent("product_viewed", productId);
  }, [productId]);
  return null;
}
