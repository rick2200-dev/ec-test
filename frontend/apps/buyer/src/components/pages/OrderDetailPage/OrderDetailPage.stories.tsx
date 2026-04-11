import type { Meta, StoryObj } from "@storybook/react";
import { OrderDetailPagePresenter } from "./OrderDetailPage.presenter";

const meta: Meta<typeof OrderDetailPagePresenter> = {
  component: OrderDetailPagePresenter,
  title: "Pages/OrderDetailPage",
};
export default meta;

type Story = StoryObj<typeof OrderDetailPagePresenter>;

const baseProps = {
  backHref: "/orders",
  backLabel: "注文履歴",
  orderId: "ord_00000000000000000000000001",
  orderIdLabel: "注文番号",
  orderedAt: "2026/04/02 09:12",
  orderedAtLabel: "注文日",
  statusHeading: "ステータス",
  statusLabel: "発送済み",
  totalLabel: "合計金額",
  totalValue: "¥12,800",
  productsLabel: "商品",
};

const contactableLine = {
  key: "sku1",
  productName: "ワイヤレスイヤホン Pro",
  skuCode: "WH-BLK-001",
  sellerName: "Tokyo Electronics",
  quantity: 1,
  unitPriceLabel: "¥12,800",
  subtotalLabel: "¥12,800",
  actionSlot: (
    <button className="inline-flex items-center rounded-md border border-blue-600 px-3 py-1.5 text-sm font-medium text-blue-600">
      出品者に問い合わせる
    </button>
  ),
};

export const Contactable: Story = {
  args: {
    ...baseProps,
    lines: [contactableLine],
  },
};

export const PendingOrder: Story = {
  args: {
    ...baseProps,
    statusLabel: "未入金",
    purchaseRequiredNotice:
      "出品者への問い合わせは、購入済み（入金後）の注文のみ可能です",
    lines: [{ ...contactableLine, actionSlot: undefined }],
  },
};
