import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Review } from "@ec-marketplace/types";
import { ReviewListPresenter } from "./ReviewList.presenter";

const baseReview: Review = {
  id: "r1",
  tenant_id: "t1",
  buyer_auth0_id: "auth0|b1",
  product_id: "p1",
  seller_id: "s1",
  product_name: "ワイヤレスイヤホン Pro",
  rating: 4,
  title: "音質がとても良い",
  body: "満足しています。",
  created_at: "2026-04-01T10:00:00Z",
  updated_at: "2026-04-01T10:00:00Z",
  reply: null,
};

const baseLabels = {
  loadMoreLabel: "もっと見る",
  loadingLabel: "読み込み中...",
  emptyLabel: "レビューはまだありません",
  replyLabel: "出品者からの返信",
  title: "レビュー",
};

describe("ReviewListPresenter", () => {
  it("renders the empty state when there are no reviews", () => {
    render(
      <ReviewListPresenter
        {...baseLabels}
        reviews={[]}
        loading={false}
        hasMore={false}
        onLoadMore={() => {}}
      />
    );
    expect(screen.getByText("レビューはまだありません")).toBeInTheDocument();
  });

  it("renders a seller reply block when reply is present", () => {
    const reviewWithReply: Review = {
      ...baseReview,
      reply: {
        id: "rp1",
        tenant_id: "t1",
        review_id: baseReview.id,
        seller_auth0_id: "auth0|s1",
        body: "ご指摘ありがとうございます。",
        created_at: "2026-04-02T15:00:00Z",
        updated_at: "2026-04-02T15:00:00Z",
      },
    };
    render(
      <ReviewListPresenter
        {...baseLabels}
        reviews={[reviewWithReply]}
        loading={false}
        hasMore={false}
        onLoadMore={() => {}}
      />
    );
    expect(screen.getByText("出品者からの返信")).toBeInTheDocument();
    expect(screen.getByText("ご指摘ありがとうございます。")).toBeInTheDocument();
  });

  it("does NOT render the reply block when reply is null", () => {
    render(
      <ReviewListPresenter
        {...baseLabels}
        reviews={[baseReview]}
        loading={false}
        hasMore={false}
        onLoadMore={() => {}}
      />
    );
    expect(screen.queryByText("出品者からの返信")).not.toBeInTheDocument();
  });
});
