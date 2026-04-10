import type { Meta, StoryObj } from "@storybook/react";
import { StatsCardPresenter } from "./StatsCard.presenter";

const meta: Meta<typeof StatsCardPresenter> = {
  title: "Seller/StatsCard",
  component: StatsCardPresenter,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div style={{ width: 300 }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof StatsCardPresenter>;

export const Default: Story = {
  args: {
    title: "今日の売上",
    value: "¥28,740",
    subtitle: "前日比 +12.5%",
  },
};

export const Success: Story = {
  args: {
    title: "今月の売上",
    value: "¥487,600",
    subtitle: "先月比 +8.3%",
    accent: "success",
  },
};

export const Warning: Story = {
  args: {
    title: "未処理注文",
    value: "2件",
    subtitle: "早めに対応してください",
    accent: "warning",
  },
};

export const Danger: Story = {
  args: {
    title: "在庫アラート",
    value: "5件",
    subtitle: "在庫が少ない商品があります",
    accent: "danger",
  },
};

export const NoSubtitle: Story = {
  args: {
    title: "Total",
    value: "128",
  },
};
