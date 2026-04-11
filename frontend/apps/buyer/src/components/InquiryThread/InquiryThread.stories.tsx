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
    senderLabel: "出品者",
    body: "お問い合わせありがとうございます。ご注文の商品は明日発送予定です。",
    timestamp: "2026/04/09 14:20",
  },
  {
    id: "m2",
    mine: true,
    senderLabel: "あなた",
    body: "ご連絡ありがとうございます！よろしくお願いいたします。",
    timestamp: "2026/04/09 14:32",
  },
];

export const Open: Story = {
  args: {
    subject: "配送時期について",
    productName: "ワイヤレスイヤホン Pro",
    skuCode: "SKU-EP-001",
    statusLabel: "対応中",
    statusTone: "open",
    messages: baseMessages,
    replyFormSlot: (
      <form className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <label className="block">
          <span className="text-sm font-medium text-gray-700">メッセージ</span>
          <textarea
            rows={4}
            className="mt-1 w-full resize-none rounded-md border border-gray-300 px-3 py-2 text-sm"
            placeholder="出品者に伝えたい内容を入力してください"
          />
        </label>
        <div className="mt-3 flex justify-end">
          <button
            type="button"
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white"
          >
            送信する
          </button>
        </div>
      </form>
    ),
  },
};

export const Closed: Story = {
  args: {
    subject: "サイズの件",
    productName: "レザーウォレット",
    skuCode: "SKU-WL-014",
    statusLabel: "完了",
    statusTone: "closed",
    threadClosedNotice: "このスレッドは完了しています",
    messages: baseMessages,
  },
};
