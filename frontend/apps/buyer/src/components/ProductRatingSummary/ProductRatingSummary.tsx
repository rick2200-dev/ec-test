"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { getProductRating } from "@/lib/api";
import { ProductRatingSummaryPresenter } from "./ProductRatingSummary.presenter";

export interface ProductRatingSummaryProps {
  productId: string;
  refreshKey?: number;
}

export default function ProductRatingSummary({ productId, refreshKey }: ProductRatingSummaryProps) {
  const t = useTranslations("reviews");
  const [averageRating, setAverageRating] = useState(0);
  const [reviewCount, setReviewCount] = useState(0);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    getProductRating(productId)
      .then((data) => {
        if (cancelled) return;
        setAverageRating(data.average_rating);
        setReviewCount(data.review_count);
        setLoaded(true);
      })
      .catch(() => {
        if (!cancelled) setLoaded(true);
      });
    return () => {
      cancelled = true;
    };
  }, [productId, refreshKey]);

  if (!loaded) return null;

  return (
    <ProductRatingSummaryPresenter
      averageRating={averageRating}
      reviewCount={reviewCount}
      ratingLabel={t("rating")}
      reviewCountLabel={t("reviewCount", { count: reviewCount })}
      noReviewsLabel={t("noReviews")}
    />
  );
}
