import type { Meta, StoryObj } from "@storybook/react";
import { OrdersPagePresenter } from "./OrdersPage.presenter";

const meta: Meta<typeof OrdersPagePresenter> = {
  component: OrdersPagePresenter,
  title: "Pages/OrdersPage",
};
export default meta;

type Story = StoryObj<typeof OrdersPagePresenter>;

const baseProps = {
  title: "注文履歴",
  description: "過去のご注文を確認できます",
  emptyLabel: "まだ注文はありません",
  orderIdLabel: "注文番号",
  orderedAtLabel: "注文日",
  totalLabel: "合計金額",
  statusLabel: "ステータス",
  productsLabel: "商品",
  actionsLabel: "操作",
  viewDetailLabel: "詳細",
};

export const Empty: Story = {
  args: { ...baseProps, orders: [] },
};

export const WithOrders: Story = {
  args: {
    ...baseProps,
    orders: [
      {
        id: "ord_1",
        href: "/orders/ord_1",
        orderedAt: "2026/04/02",
        totalLabel: "¥12,800",
        statusLabel: "発送済み",
        statusTone: "shipped",
        productSummary: "ワイヤレスイヤホン Pro x1",
      },
      {
        id: "ord_2",
        href: "/orders/ord_2",
        orderedAt: "2026/04/09",
        totalLabel: "¥5,980",
        statusLabel: "未入金",
        statusTone: "pending",
        productSummary: "USB-Cハブ x1",
      },
    ],
  },
};
