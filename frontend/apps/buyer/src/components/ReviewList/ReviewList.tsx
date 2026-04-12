"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import type { Review } from "@ec-marketplace/types";
import { listProductReviews } from "@/lib/api";
import { ReviewListPresenter } from "./ReviewList.presenter";

const PAGE_SIZE = 5;

export interface ReviewListProps {
  productId: string;
  refreshKey?: number;
}

export default function ReviewList({ productId, refreshKey }: ReviewListProps) {
  const t = useTranslations("reviews");
  const [reviews, setReviews] = useState<Review[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);

  const load = useCallback(
    async (currentOffset: number) => {
      setLoading(true);
      try {
        const data = await listProductReviews(productId, {
          limit: PAGE_SIZE,
          offset: currentOffset,
        });
        setReviews((prev) =>
          currentOffset === 0 ? data.items : [...prev, ...data.items],
        );
        setTotal(data.total);
      } catch {
        // Silently fail — the section simply stays empty.
      } finally {
        setLoading(false);
      }
    },
    [productId],
  );

  useEffect(() => {
    setOffset(0);
    load(0);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- refreshKey triggers a full reload
  }, [load, refreshKey]);

  const handleLoadMore = () => {
    const nextOffset = offset + PAGE_SIZE;
    setOffset(nextOffset);
    load(nextOffset);
  };

  return (
    <ReviewListPresenter
      reviews={reviews}
      total={total}
      loading={loading}
      hasMore={reviews.length < total}
      onLoadMore={handleLoadMore}
      loadMoreLabel={t("loadMore")}
      loadingLabel={t("loading")}
      emptyLabel={t("noReviews")}
      replyLabel={t("sellerReply")}
      title={t("title")}
    />
  );
}
