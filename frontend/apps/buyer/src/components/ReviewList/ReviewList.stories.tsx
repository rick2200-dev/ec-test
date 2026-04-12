import type { Meta, StoryObj } from "@storybook/react";
import { ReviewListPresenter } from "./ReviewList.presenter";

const meta: Meta<typeof ReviewListPresenter> = {
  component: ReviewListPresenter,
  title: "Reviews/ReviewList",
};
export default meta;

type Story = StoryObj<typeof ReviewListPresenter>;

const sampleReview = {
  id: "r1",
  tenant_id: "t1",
  buyer_auth0_id: "auth0|buyer1",
  product_id: "p1",
  seller_id: "s1",
  product_name: "ワイヤレスイヤホン Pro",
  rating: 4,
  title: "音質がとても良い",
  body: "ノイズキャンセリングの性能も高く、満足しています。",
  created_at: "2025-10-01T10:00:00Z",
  updated_at: "2025-10-01T10:00:00Z",
  reply: null,
};

const reviewWithReply = {
  ...sampleReview,
  id: "r2",
  rating: 3,
  title: "バッテリーの持ちが気になる",
  body: "音質は良いですが、バッテリーが4時間しか持ちません。",
  reply: {
    id: "rp1",
    tenant_id: "t1",
    review_id: "r2",
    seller_auth0_id: "auth0|seller1",
    body: "ご指摘ありがとうございます。ファームウェアアップデートでバッテリー持続時間の改善を予定しています。",
    created_at: "2025-10-02T15:00:00Z",
    updated_at: "2025-10-02T15:00:00Z",
  },
};

const base = {
  loadMoreLabel: "もっと見る",
  loadingLabel: "読み込み中...",
  emptyLabel: "レビューはまだありません",
  replyLabel: "出品者からの返信",
  title: "レビュー",
};

export const WithReviews: Story = {
  args: {
    ...base,
    reviews: [sampleReview, reviewWithReply],
    loading: false,
    hasMore: true,
    onLoadMore: () => {},
  },
};

export const Empty: Story = {
  args: {
    ...base,
    reviews: [],
    loading: false,
    hasMore: false,
    onLoadMore: () => {},
  },
};

export const Loading: Story = {
  args: {
    ...base,
    reviews: [],
    loading: true,
    hasMore: false,
    onLoadMore: () => {},
  },
};
