import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import {
  NotificationsPagePresenter,
  type NotificationCardItem,
  type NotificationFilter,
} from "./NotificationsPage.presenter";

const meta: Meta<typeof NotificationsPagePresenter> = {
  title: "Seller/Pages/NotificationsPage",
  component: NotificationsPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof NotificationsPagePresenter>;

const labels = {
  title: "通知",
  description: "注文・在庫・問い合わせに関するお知らせを確認できます",
  emptyMessage: "通知はまだありません",
  unreadBadgeLabel: "未読",
  markAllReadLabel: "すべて既読にする",
  filterAllLabel: "すべて",
  filterUnreadLabel: "未読のみ",
};

const sampleNotifications: NotificationCardItem[] = [
  {
    id: "s-1",
    createdAtLabel: "2026/04/12 08:44",
    title: "新しい注文が入りました",
    body: "注文 #ord-42 が作成されました。金額: ¥14,080",
    category: "order",
    unread: true,
    href: "/orders",
  },
  {
    id: "s-2",
    createdAtLabel: "2026/04/11 21:05",
    title: "在庫が少なくなっています",
    body: "SKU SKU-RED-M（Premium T-Shirt）の在庫が 3 個になりました。",
    category: "inventory",
    unread: true,
    href: "/inventory",
  },
  {
    id: "s-3",
    createdAtLabel: "2026/04/11 13:18",
    title: "購入者から問い合わせが届きました",
    body: "購入者から「発送時期について」について問い合わせがあります。",
    category: "inquiry",
    unread: true,
    href: "/inquiries",
  },
  {
    id: "s-4",
    createdAtLabel: "2026/04/10 09:30",
    title: "注文の支払いが完了しました",
    body: "注文 #ord-41 の支払いが確認されました。準備を開始してください。",
    category: "order",
    unread: false,
    href: "/orders",
  },
  {
    id: "s-5",
    createdAtLabel: "2026/03/28 12:00",
    title: "出店が承認されました",
    body: "出店申請が承認されました。商品登録を開始できます。",
    category: "account",
    unread: false,
  },
];

const Interactive = ({
  initial,
}: {
  initial: NotificationCardItem[];
}) => {
  const [items, setItems] = useState(initial);
  const [filter, setFilter] = useState<NotificationFilter>("all");

  const visible = filter === "unread" ? items.filter((i) => i.unread) : items;

  return (
    <NotificationsPagePresenter
      {...labels}
      filter={filter}
      onFilterChange={setFilter}
      onMarkAllRead={() => setItems((prev) => prev.map((i) => ({ ...i, unread: false })))}
      notifications={visible}
    />
  );
};

export const Default: Story = {
  render: () => <Interactive initial={sampleNotifications} />,
};

export const Empty: Story = {
  render: () => <Interactive initial={[]} />,
};

export const AllRead: Story = {
  render: () => (
    <Interactive initial={sampleNotifications.map((n) => ({ ...n, unread: false }))} />
  ),
};
