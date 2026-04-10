import type { Meta, StoryObj } from "@storybook/react";
import { StatusBadgePresenter } from "./StatusBadge.presenter";

const meta: Meta<typeof StatusBadgePresenter> = {
  title: "Admin/StatusBadge",
  component: StatusBadgePresenter,
  parameters: { layout: "centered" },
};

export default meta;
type Story = StoryObj<typeof StatusBadgePresenter>;

export const Success: Story = {
  args: { tone: "success", label: "有効" },
};

export const Warning: Story = {
  args: { tone: "warning", label: "承認待ち" },
};

export const Danger: Story = {
  args: { tone: "danger", label: "停止中" },
};

export const Neutral: Story = {
  args: { tone: "neutral", label: "Unknown" },
};
