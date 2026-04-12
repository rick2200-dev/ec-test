"use client";

import { StarRatingPresenter } from "@/components/StarRating/StarRating.presenter";

export interface ProductRatingSummaryPresenterProps {
  averageRating: number;
  reviewCount: number;
  ratingLabel: string;
  reviewCountLabel: string;
  noReviewsLabel: string;
}

export function ProductRatingSummaryPresenter({
  averageRating,
  reviewCount,
  ratingLabel,
  reviewCountLabel,
  noReviewsLabel,
}: ProductRatingSummaryPresenterProps) {
  if (reviewCount === 0) {
    return (
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <StarRatingPresenter rating={0} size="sm" />
        <span>{noReviewsLabel}</span>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2" aria-label={ratingLabel}>
      <StarRatingPresenter rating={averageRating} size="sm" />
      <span className="text-sm font-medium text-gray-900">{averageRating.toFixed(1)}</span>
      <span className="text-sm text-gray-500">({reviewCountLabel})</span>
    </div>
  );
}
