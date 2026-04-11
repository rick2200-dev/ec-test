import type { Meta, StoryObj } from "@storybook/react";
import { OrderDetailPagePresenter } from "./OrderDetailPage.presenter";

const meta: Meta<typeof OrderDetailPagePresenter> = {
  title: "Buyer/Pages/OrderDetailPage",
  component: OrderDetailPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof OrderDetailPagePresenter>;

const baseLabels = {
  quantity: "数量",
  deletedBadge: "削除済み",
  imageMissing: "画像なし",
};

const baseArgs = {
  title: "注文詳細",
  backLabel: "注文履歴に戻る",
  backHref: "/orders",
  orderNumberLabel: "注文番号: ord-12345",
  orderedAtLabel: "注文日",
  orderedAtValue: "2026年4月8日",
  sellerLabel: "販売者",
  sellerName: "Acme Audio",
  statusLabel: "発送済み",
  itemsLabel: "注文商品",
  subtotalLabel: "小計",
  subtotalValue: "¥13,300",
  shippingFeeLabel: "送料",
  shippingFeeValue: "¥500",
  totalLabel: "合計",
  totalValue: "¥13,800",
};

export const AllLive: Story = {
  args: {
    ...baseArgs,
    lines: [
      {
        id: "l1",
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
      {
        id: "l2",
        productName: "USB-C Cable",
        skuCode: "USBC-1M",
        quantity: 1,
        unitPriceLabel: "¥500",
        lineTotalLabel: "¥500",
        imageUrl: "",
        productHref: "/products/usb-c-cable",
        isDeleted: false,
        labels: baseLabels,
      },
    ],
  },
};

export const MixedWithDeleted: Story = {
  args: {
    ...baseArgs,
    lines: [
      {
        id: "l1",
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
      {
        id: "l2",
        productName: "Vintage Leather Wallet (販売終了)",
        skuCode: "VLW-BRN",
        quantity: 1,
        unitPriceLabel: "¥500",
        lineTotalLabel: "¥500",
        imageUrl: "",
        productHref: "",
        isDeleted: true,
        labels: baseLabels,
      },
    ],
  },
};

export const AllDeleted: Story = {
  args: {
    ...baseArgs,
    sellerName: "不明な販売者",
    lines: [
      {
        id: "l1",
        productName: "Discontinued Item A",
        skuCode: "DISC-A",
        quantity: 1,
        unitPriceLabel: "¥8,000",
        lineTotalLabel: "¥8,000",
        imageUrl: "",
        productHref: "",
        isDeleted: true,
        labels: baseLabels,
      },
      {
        id: "l2",
        productName: "Discontinued Item B",
        skuCode: "DISC-B",
        quantity: 2,
        unitPriceLabel: "¥2,500",
        lineTotalLabel: "¥5,000",
        imageUrl: "",
        productHref: "",
        isDeleted: true,
        labels: baseLabels,
      },
    ],
  },
};
