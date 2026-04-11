import type { Meta, StoryObj } from "@storybook/react";
import { OrderHistoryPagePresenter } from "./OrderHistoryPage.presenter";

const meta: Meta<typeof OrderHistoryPagePresenter> = {
  title: "Buyer/Pages/OrderHistoryPage",
  component: OrderHistoryPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof OrderHistoryPagePresenter>;

const baseArgs = {
  title: "購入履歴",
  sellerLabel: "販売者",
  totalLabel: "合計",
  emptyMessage: "まだ購入履歴はありません",
  browseCtaLabel: "商品を探す",
  browseCtaHref: "/products",
};

export const Default: Story = {
  args: {
    ...baseArgs,
    orders: [
      {
        id: "ord-1",
        href: "/orders/ord-1",
        createdAtLabel: "2026年4月8日",
        sellerName: "Acme Audio",
        statusLabel: "発送済み",
        totalLabel: "¥14,080",
      },
      {
        id: "ord-2",
        href: "/orders/ord-2",
        createdAtLabel: "2026年3月22日",
        sellerName: "Studio Leather",
        statusLabel: "配達済み",
        totalLabel: "¥26,400",
      },
      {
        id: "ord-3",
        href: "/orders/ord-3",
        createdAtLabel: "2026年3月15日",
        sellerName: "Pottery Lab",
        statusLabel: "完了",
        totalLabel: "¥3,520",
      },
    ],
  },
};

export const Empty: Story = {
  args: {
    ...baseArgs,
    orders: [],
  },
};

export const UnknownSeller: Story = {
  args: {
    ...baseArgs,
    orders: [
      {
        id: "ord-legacy",
        href: "/orders/ord-legacy",
        createdAtLabel: "2025年8月1日",
        sellerName: "不明な販売者",
        statusLabel: "完了",
        totalLabel: "¥2,200",
      },
    ],
  },
};
