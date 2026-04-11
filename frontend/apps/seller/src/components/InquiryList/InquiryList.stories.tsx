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
  description: "購入者からのお問い合わせを一覧で確認できます",
  emptyLabel: "お問い合わせはまだありません",
  productColumnLabel: "商品",
  lastMessageColumnLabel: "最終メッセージ",
  statusColumnLabel: "ステータス",
  unreadLabel: "未読",
};

export const Empty: Story = {
  args: { ...baseProps, items: [] },
};

export const WithThreads: Story = {
  args: {
    ...baseProps,
    items: [
      {
        id: "1",
        href: "/inquiries/1",
        productName: "オーガニックコットンTシャツ",
        skuCode: "OCT-WHT-M",
        subject: "サイズ交換について",
        lastMessageAt: "2026/04/09 14:32",
        status: "open",
        statusLabel: "対応中",
        unreadCount: 3,
      },
      {
        id: "2",
        href: "/inquiries/2",
        productName: "デニムジャケット",
        skuCode: "DJ-M-IND",
        subject: "発送予定日",
        lastMessageAt: "2026/04/05 09:12",
        status: "closed",
        statusLabel: "完了",
        unreadCount: 0,
      },
    ],
  },
};
