import type { Meta, StoryObj } from "@storybook/react";
import { CartBadgePresenter } from "./CartBadge.presenter";

const meta: Meta<typeof CartBadgePresenter> = {
  component: CartBadgePresenter,
  title: "Buyer/CartBadge",
};
export default meta;

type Story = StoryObj<typeof CartBadgePresenter>;

export const Empty: Story = {
  args: { count: 0, ariaLabel: "カート" },
};

export const FewItems: Story = {
  args: { count: 3, ariaLabel: "カート" },
};

export const Many: Story = {
  args: { count: 150, ariaLabel: "カート" },
};
