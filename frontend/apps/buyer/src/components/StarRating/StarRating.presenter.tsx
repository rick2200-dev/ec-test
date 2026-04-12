"use client";

import { useId } from "react";

export interface StarRatingPresenterProps {
  rating: number;
  maxStars?: number;
  size?: "sm" | "md" | "lg";
}

const sizeMap = { sm: "w-4 h-4", md: "w-5 h-5", lg: "w-6 h-6" } as const;

export function StarRatingPresenter({
  rating,
  maxStars = 5,
  size = "md",
}: StarRatingPresenterProps) {
  const cls = sizeMap[size];
  const uid = useId();
  return (
    <span className="inline-flex items-center gap-0.5" aria-label={`${rating} / ${maxStars}`}>
      {Array.from({ length: maxStars }, (_, i) => {
        const fill = Math.min(Math.max(rating - i, 0), 1);
        return (
          <svg
            key={i}
            className={`${cls} ${fill >= 1 ? "text-yellow-400" : fill > 0 ? "text-yellow-400" : "text-gray-300"}`}
            viewBox="0 0 20 20"
            fill="currentColor"
            aria-hidden="true"
          >
            <defs>
              {fill > 0 && fill < 1 && (
                <linearGradient id={`star-grad-${uid}-${i}`}>
                  <stop offset={`${fill * 100}%`} stopColor="currentColor" />
                  <stop offset={`${fill * 100}%`} stopColor="#d1d5db" />
                </linearGradient>
              )}
            </defs>
            <path
              fill={fill > 0 && fill < 1 ? `url(#star-grad-${uid}-${i})` : undefined}
              fillRule="evenodd"
              d="M10.868 2.884c-.321-.772-1.415-.772-1.736 0l-1.83 4.401-4.753.381c-.833.067-1.171 1.107-.536 1.651l3.62 3.102-1.106 4.637c-.194.813.691 1.456 1.405 1.02L10 15.591l4.069 2.485c.713.436 1.598-.207 1.404-1.02l-1.106-4.637 3.62-3.102c.635-.544.297-1.584-.536-1.65l-4.752-.382-1.831-4.401Z"
              clipRule="evenodd"
            />
          </svg>
        );
      })}
    </span>
  );
}
