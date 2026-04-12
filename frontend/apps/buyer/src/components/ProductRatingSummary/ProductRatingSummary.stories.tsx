import type { Meta, StoryObj } from "@storybook/react";
import { ProductRatingSummaryPresenter } from "./ProductRatingSummary.presenter";

const meta: Meta<typeof ProductRatingSummaryPresenter> = {
  component: ProductRatingSummaryPresenter,
  title: "Reviews/ProductRatingSummary",
};
export default meta;

type Story = StoryObj<typeof ProductRatingSummaryPresenter>;

export const WithReviews: Story = {
  args: {
    averageRating: 4.3,
    reviewCount: 28,
    ratingLabel: "評価",
    reviewCountLabel: "28件のレビュー",
    noReviewsLabel: "レビューはまだありません",
  },
};

export const NoReviews: Story = {
  args: {
    averageRating: 0,
    reviewCount: 0,
    ratingLabel: "評価",
    reviewCountLabel: "0件のレビュー",
    noReviewsLabel: "レビューはまだありません",
  },
};

export const PerfectScore: Story = {
  args: {
    averageRating: 5.0,
    reviewCount: 3,
    ratingLabel: "評価",
    reviewCountLabel: "3件のレビュー",
    noReviewsLabel: "レビューはまだありません",
  },
};
