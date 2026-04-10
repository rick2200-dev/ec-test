import type { Meta, StoryObj } from "@storybook/react";
import { AdminDashboardPagePresenter } from "./DashboardPage.presenter";

const meta: Meta<typeof AdminDashboardPagePresenter> = {
  title: "Admin/Pages/DashboardPage",
  component: AdminDashboardPagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof AdminDashboardPagePresenter>;

const sampleArgs = {
  heading: {
    title: "プラットフォーム管理",
    description: "テナントとセラーの状況を確認できます",
  },
  statsCards: [
    { id: "tenants", title: "総テナント数", value: "12", subtitle: "前月比 +2" },
    { id: "sellers", title: "総セラー数", value: "248", subtitle: "前月比 +18" },
    {
      id: "transactions",
      title: "月間取引額",
      value: "¥18,400,000",
      subtitle: "前月比 +15.2%",
    },
    {
      id: "commission",
      title: "月間手数料収入",
      value: "¥920,000",
      subtitle: "前月比 +15.2%",
    },
  ],
  pendingSection: {
    title: "セラー申請",
    viewAllHref: "/sellers",
    viewAllLabel: "すべて表示",
    columnLabels: {
      sellerName: "セラー名",
      tenant: "テナント",
      applicationDate: "申請日",
      status: "ステータス",
    },
    rows: [
      {
        id: "s1",
        name: "Kyoto Crafts",
        tenantName: "Default Tenant",
        createdAtLabel: "2025-04-08",
        badge: { tone: "warning" as const, label: "承認待ち" },
      },
      {
        id: "s2",
        name: "Hokkaido Foods",
        tenantName: "Default Tenant",
        createdAtLabel: "2025-04-09",
        badge: { tone: "warning" as const, label: "承認待ち" },
      },
    ],
  },
  serviceHealthSection: {
    title: "サービス状態",
    services: [
      { name: "API Gateway", badge: { tone: "success" as const, label: "正常" } },
      { name: "認証サービス", badge: { tone: "success" as const, label: "正常" } },
      { name: "決済サービス", badge: { tone: "success" as const, label: "正常" } },
      { name: "検索サービス", badge: { tone: "warning" as const, label: "低下" } },
      { name: "通知サービス", badge: { tone: "success" as const, label: "正常" } },
      { name: "画像処理", badge: { tone: "danger" as const, label: "停止" } },
    ],
  },
};

export const Default: Story = {
  args: sampleArgs,
};

export const Healthy: Story = {
  args: {
    ...sampleArgs,
    serviceHealthSection: {
      ...sampleArgs.serviceHealthSection,
      services: sampleArgs.serviceHealthSection.services.map((s) => ({
        ...s,
        badge: { tone: "success" as const, label: "正常" },
      })),
    },
    pendingSection: { ...sampleArgs.pendingSection, rows: [] },
  },
};
