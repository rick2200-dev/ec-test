import type { Meta, StoryObj } from "@storybook/react";
import { OrderLineItemPresenter } from "./OrderLineItem.presenter";

const meta: Meta<typeof OrderLineItemPresenter> = {
  title: "Buyer/OrderLineItem",
  component: OrderLineItemPresenter,
  parameters: { layout: "padded" },
  decorators: [
    (Story) => (
      <ul className="max-w-2xl divide-y divide-gray-200 rounded-lg border border-gray-200 bg-white px-4">
        <Story />
      </ul>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof OrderLineItemPresenter>;

const baseLabels = {
  quantity: "数量",
  deletedBadge: "削除済み",
  imageMissing: "画像なし",
};

export const LiveWithImage: Story = {
  args: {
    productName: "Wireless Headphones",
    skuCode: "WH-001-BLK",
    quantity: 1,
    unitPriceLabel: "¥12,800",
    lineTotalLabel: "¥12,800",
    imageUrl: "https://picsum.photos/seed/headphones/200/200",
    productHref: "/products/wireless-headphones",
    isDeleted: false,
    labels: baseLabels,
  },
};

export const LiveWithoutImage: Story = {
  args: {
    ...LiveWithImage.args!,
    imageUrl: "",
  },
};

export const Deleted: Story = {
  args: {
    ...LiveWithImage.args!,
    imageUrl: "",
    productHref: "",
    isDeleted: true,
  },
};

export const DeletedWithStaleImage: Story = {
  args: {
    ...LiveWithImage.args!,
    imageUrl: "",
    productHref: "",
    isDeleted: true,
    productName: "販売終了した商品 - Vintage Leather Wallet",
  },
};
