"use client";

import type { Review } from "@ec-marketplace/types";
import { StarRatingPresenter } from "@/components/StarRating/StarRating.presenter";

export interface ReviewListPresenterProps {
  reviews: Review[];
  total: number;
  loading: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
  loadMoreLabel: string;
  loadingLabel: string;
  emptyLabel: string;
  replyLabel: string;
  title: string;
}

export function ReviewListPresenter({
  reviews,
  total,
  loading,
  hasMore,
  onLoadMore,
  loadMoreLabel,
  loadingLabel,
  emptyLabel,
  replyLabel,
  title,
}: ReviewListPresenterProps) {
  return (
    <section>
      <h2 className="text-lg font-semibold text-gray-900">{title}</h2>

      {reviews.length === 0 && !loading && (
        <p className="mt-3 text-sm text-gray-500">{emptyLabel}</p>
      )}

      <div className="mt-4 space-y-4">
        {reviews.map((review) => (
          <article key={review.id} className="rounded-lg border border-gray-200 p-4">
            <div className="flex items-center gap-2">
              <StarRatingPresenter rating={review.rating} size="sm" />
              <span className="text-sm font-medium text-gray-900">{review.title}</span>
            </div>
            <p className="mt-2 text-sm text-gray-700 whitespace-pre-wrap">{review.body}</p>
            <p className="mt-2 text-xs text-gray-400">
              {new Date(review.created_at).toLocaleDateString()}
            </p>

            {review.reply && (
              <div className="mt-3 rounded-md bg-gray-50 p-3 border-l-2 border-blue-400">
                <p className="text-xs font-medium text-blue-600">{replyLabel}</p>
                <p className="mt-1 text-sm text-gray-700 whitespace-pre-wrap">
                  {review.reply.body}
                </p>
                <p className="mt-1 text-xs text-gray-400">
                  {new Date(review.reply.created_at).toLocaleDateString()}
                </p>
              </div>
            )}
          </article>
        ))}
      </div>

      {loading && (
        <p className="mt-4 text-center text-sm text-gray-500">{loadingLabel}</p>
      )}

      {hasMore && !loading && (
        <div className="mt-4 text-center">
          <button
            type="button"
            onClick={onLoadMore}
            className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {loadMoreLabel}
          </button>
        </div>
      )}
    </section>
  );
}
