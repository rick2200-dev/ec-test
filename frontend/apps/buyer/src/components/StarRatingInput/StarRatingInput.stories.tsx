import type { Meta, StoryObj } from "@storybook/react";
import { StarRatingInputPresenter } from "./StarRatingInput.presenter";

const meta: Meta<typeof StarRatingInputPresenter> = {
  component: StarRatingInputPresenter,
  title: "Reviews/StarRatingInput",
};
export default meta;

type Story = StoryObj<typeof StarRatingInputPresenter>;

const base = {
  onChange: () => {},
  onHover: () => {},
  hoverValue: null,
};

export const Empty: Story = { args: { ...base, value: 0, size: "md" } };
export const ThreeStars: Story = { args: { ...base, value: 3, size: "md" } };
export const HoverFour: Story = { args: { ...base, value: 2, hoverValue: 4, size: "md" } };
export const Disabled: Story = { args: { ...base, value: 4, size: "md", disabled: true } };
export const Large: Story = { args: { ...base, value: 5, size: "lg" } };
