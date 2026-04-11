import type { Meta, StoryObj } from "@storybook/react";
import { InquiryListPresenter } from "./InquiryList.presenter";

const meta: Meta<typeof InquiryListPresenter> = {
  component: InquiryListPresenter,
  title: "Inquiries/InquiryList",
};
export default meta;

type Story = StoryObj<typeof InquiryListPresenter>;

const baseProps = {
  title: "お問い合わせ",
  description: "購入した商品について出品者に問い合わせたスレッドの一覧です",
  emptyLabel: "お問い合わせはまだありません",
  productColumnLabel: "商品",
  lastMessageColumnLabel: "最終メッセージ",
  statusColumnLabel: "ステータス",
  unreadLabel: "未読",
};

export const Empty: Story = {
  args: {
    ...baseProps,
    items: [],
  },
};

export const WithThreads: Story = {
  args: {
    ...baseProps,
    items: [
      {
        id: "1",
        href: "/inquiries/1",
        productName: "ワイヤレスイヤホン Pro",
        skuCode: "SKU-EP-001",
        subject: "配送時期について",
        lastMessageAt: "2026/04/09 14:32",
        status: "open",
        statusLabel: "対応中",
        unreadCount: 2,
      },
      {
        id: "2",
        href: "/inquiries/2",
        productName: "レザーウォレット",
        skuCode: "SKU-WL-014",
        subject: "サイズの件",
        lastMessageAt: "2026/04/05 09:12",
        status: "closed",
        statusLabel: "完了",
        unreadCount: 0,
      },
    ],
  },
};
