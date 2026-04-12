import type { Meta, StoryObj } from "@storybook/react";
import { StarRatingPresenter } from "./StarRating.presenter";

const meta: Meta<typeof StarRatingPresenter> = {
  component: StarRatingPresenter,
  title: "Reviews/StarRating",
};
export default meta;

type Story = StoryObj<typeof StarRatingPresenter>;

export const Full: Story = { args: { rating: 5, size: "md" } };
export const Half: Story = { args: { rating: 3.5, size: "md" } };
export const Empty: Story = { args: { rating: 0, size: "md" } };
export const Small: Story = { args: { rating: 4, size: "sm" } };
export const Large: Story = { args: { rating: 4.2, size: "lg" } };
