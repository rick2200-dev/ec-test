import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Review } from "@ec-marketplace/types";
import { SellerReviewsListPresenter } from "./SellerReviewsList.presenter";

const labels = {
  loadingLabel: "読み込み中...",
  emptyLabel: "レビューはまだありません",
  noReplyBadge: "未返信",
  hasReplyBadge: "返信済み",
  columns: {
    product: "商品",
    rating: "評価",
    title: "内容",
    date: "投稿日",
    state: "状態",
    actions: "操作",
  },
};

const baseReview: Review = {
  id: "r1",
  tenant_id: "t1",
  buyer_auth0_id: "auth0|b1",
  product_id: "p1",
  seller_id: "s1",
  product_name: "商品A",
  rating: 4,
  title: "良かった",
  body: "満足しています",
  created_at: "2026-04-01T10:00:00Z",
  updated_at: "2026-04-01T10:00:00Z",
  reply: null,
};

describe("SellerReviewsListPresenter", () => {
  it("renders the loading state", () => {
    render(
      <SellerReviewsListPresenter
        {...labels}
        reviews={[]}
        loading
        loaded={false}
        locale="ja"
        renderRow={() => null}
      />
    );
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
  });

  it("renders the empty state", () => {
    render(
      <SellerReviewsListPresenter
        {...labels}
        reviews={[]}
        loading={false}
        loaded
        locale="ja"
        renderRow={() => null}
      />
    );
    expect(screen.getByText("レビューはまだありません")).toBeInTheDocument();
  });

  it("renders reviews and the no-reply / replied badges", () => {
    const reviewWithReply: Review = {
      ...baseReview,
      id: "r2",
      reply: {
        id: "rp1",
        tenant_id: "t1",
        review_id: "r2",
        seller_auth0_id: "auth0|s1",
        body: "ありがとうございます。",
        created_at: "2026-04-02T10:00:00Z",
        updated_at: "2026-04-02T10:00:00Z",
      },
    };
    render(
      <SellerReviewsListPresenter
        {...labels}
        reviews={[baseReview, reviewWithReply]}
        loading={false}
        loaded
        locale="ja"
        renderRow={(r) => <span>{r.reply ? "edit" : "reply"}</span>}
      />
    );
    expect(screen.getByText("未返信")).toBeInTheDocument();
    expect(screen.getByText("返信済み")).toBeInTheDocument();
    expect(screen.getByText("ありがとうございます。")).toBeInTheDocument();
  });
});
