"use client";

import { StarRatingPresenter } from "./StarRating.presenter";

export interface StarRatingProps {
  rating: number;
  maxStars?: number;
  size?: "sm" | "md" | "lg";
}

export default function StarRating({ rating, maxStars, size }: StarRatingProps) {
  return <StarRatingPresenter rating={rating} maxStars={maxStars} size={size} />;
}
