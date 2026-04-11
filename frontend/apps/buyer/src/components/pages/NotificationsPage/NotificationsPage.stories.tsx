import type { Meta, StoryObj } from "@storybook/react";
import { NotificationsPagePresenter } from "./NotificationsPage.presenter";

const meta: Meta<typeof NotificationsPagePresenter> = {
  title: "Buyer/Pages/NotificationsPage",
  component: NotificationsPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof NotificationsPagePresenter>;

const baseArgs = {
  title: "通知",
  description: "注文や問い合わせに関するお知らせの一覧です",
  emptyMessage: "通知はまだありません",
  unreadBadgeLabel: "未読",
};

export const Default: Story = {
  args: {
    ...baseArgs,
    notifications: [
      {
        id: "ntf-1",
        createdAtLabel: "2026年4月11日 18:12",
        title: "商品が発送されました",
        body: "ご注文 #ord-1 の商品を発送しました。追跡番号: JP1234567890",
        href: "/orders/ord-1",
        unread: true,
      },
      {
        id: "ntf-2",
        createdAtLabel: "2026年4月10日 15:40",
        title: "出品者から返信があります",
        body: "「発送時期について」のスレッドに新しい返信があります。",
        href: "/inquiries",
        unread: true,
      },
      {
        id: "ntf-3",
        createdAtLabel: "2026年4月9日 11:05",
        title: "注文の支払いが完了しました",
        body: "ご注文 #ord-2 の支払いが確認されました。",
        href: "/orders/ord-2",
        unread: false,
      },
    ],
  },
};

export const Empty: Story = {
  args: {
    ...baseArgs,
    notifications: [],
  },
};

export const AllRead: Story = {
  args: {
    ...baseArgs,
    notifications: [
      {
        id: "ntf-old-1",
        createdAtLabel: "2026年4月3日 18:22",
        title: "商品が配達されました",
        body: "ご注文 #ord-3 の商品がお届け先に到着しました。",
        href: "/orders/ord-3",
        unread: false,
      },
    ],
  },
};
