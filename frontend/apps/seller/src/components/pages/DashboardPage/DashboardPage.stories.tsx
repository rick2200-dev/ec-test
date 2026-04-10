import type { Meta, StoryObj } from "@storybook/react";
import { DashboardPagePresenter } from "./DashboardPage.presenter";

const meta: Meta<typeof DashboardPagePresenter> = {
  title: "Seller/Pages/DashboardPage",
  component: DashboardPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof DashboardPagePresenter>;

const sampleArgs = {
  heading: {
    title: "ダッシュボード",
    description: "売上と注文の概要",
  },
  statsCards: [
    {
      id: "today",
      title: "今日の売上",
      value: "¥28,740",
      subtitle: "前日比 +12.5%",
      accent: "success" as const,
    },
    {
      id: "month",
      title: "今月の売上",
      value: "¥487,600",
      subtitle: "先月比 +8.3%",
      accent: "default" as const,
    },
    {
      id: "pending",
      title: "未処理注文",
      value: "2件",
      subtitle: "早めに対応してください",
      accent: "warning" as const,
    },
    {
      id: "alerts",
      title: "在庫アラート",
      value: "5件",
      subtitle: "在庫が少ない商品があります",
      accent: "danger" as const,
    },
  ],
  recentOrdersSection: {
    title: "最近の注文",
    viewAllHref: "/orders",
    viewAllLabel: "すべて表示",
    columnLabels: {
      orderId: "注文番号",
      buyer: "購入者",
      amount: "金額",
      status: "ステータス",
      date: "日時",
    },
    orders: [
      {
        id: "ORD-001",
        buyerName: "山田太郎",
        amountLabel: "¥4,800",
        statusLabel: "未処理",
        statusClassName: "bg-yellow-100 text-yellow-800",
        dateLabel: "4月10日 10:23",
      },
      {
        id: "ORD-002",
        buyerName: "佐藤花子",
        amountLabel: "¥12,400",
        statusLabel: "発送済み",
        statusClassName: "bg-purple-100 text-purple-800",
        dateLabel: "4月9日 18:45",
      },
      {
        id: "ORD-003",
        buyerName: "鈴木一郎",
        amountLabel: "¥3,200",
        statusLabel: "完了",
        statusClassName: "bg-green-100 text-green-800",
        dateLabel: "4月9日 12:11",
      },
    ],
  },
};

export const Default: Story = {
  args: sampleArgs,
};

export const NoOrders: Story = {
  args: {
    ...sampleArgs,
    recentOrdersSection: { ...sampleArgs.recentOrdersSection, orders: [] },
  },
};
