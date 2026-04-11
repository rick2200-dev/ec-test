import type { Meta, StoryObj } from "@storybook/react";
import { InquiryThreadPresenter } from "./InquiryThread.presenter";

const meta: Meta<typeof InquiryThreadPresenter> = {
  component: InquiryThreadPresenter,
  title: "Inquiries/InquiryThread",
};
export default meta;

type Story = StoryObj<typeof InquiryThreadPresenter>;

const baseMessages = [
  {
    id: "m1",
    mine: false,
    senderLabel: "購入者",
    body: "発送はいつ頃になりますか？",
    timestamp: "2026/04/09 14:20",
  },
  {
    id: "m2",
    mine: true,
    senderLabel: "あなた",
    body: "明日発送を予定しております。よろしくお願いいたします。",
    timestamp: "2026/04/09 14:32",
  },
];

export const Open: Story = {
  args: {
    subject: "発送予定について",
    productName: "オーガニックコットンTシャツ",
    skuCode: "OCT-WHT-M",
    buyerLabel: "購入者: auth0|abc123",
    statusLabel: "対応中",
    statusTone: "open",
    messages: baseMessages,
    closeButtonSlot: (
      <button className="rounded-md border border-border bg-white px-3 py-1 text-xs font-medium text-text-primary">
        スレッドを閉じる
      </button>
    ),
  },
};

export const Closed: Story = {
  args: {
    subject: "発送予定について",
    productName: "オーガニックコットンTシャツ",
    skuCode: "OCT-WHT-M",
    buyerLabel: "購入者: auth0|abc123",
    statusLabel: "完了",
    statusTone: "closed",
    threadClosedNotice: "このスレッドは完了しています",
    messages: baseMessages,
  },
};
