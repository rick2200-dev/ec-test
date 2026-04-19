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
    title: "Dashboard",
    description: "Platform overview",
  },
  statsCards: [
    { id: "sellers", title: "Total Sellers", value: "248", subtitle: "+18 MoM" },
    {
      id: "transactions",
      title: "Monthly Transaction Volume",
      value: "¥18,400,000",
      subtitle: "+15.2% MoM",
    },
    {
      id: "commission",
      title: "Monthly Commission Revenue",
      value: "¥920,000",
      subtitle: "+15.2% MoM",
    },
  ],
  pendingSection: {
    title: "Pending Seller Applications",
    viewAllHref: "/sellers",
    viewAllLabel: "View all",
    emptyLabel: "No pending applications",
    columnLabels: {
      sellerName: "Seller Name",
      applicationDate: "Application Date",
      status: "Status",
    },
    rows: [
      {
        id: "s1",
        name: "Kyoto Crafts",
        createdAtLabel: "2025-04-08",
        badge: { tone: "warning" as const, label: "Pending" },
      },
      {
        id: "s2",
        name: "Hokkaido Foods",
        createdAtLabel: "2025-04-09",
        badge: { tone: "warning" as const, label: "Pending" },
      },
    ],
  },
};

export const Default: Story = {
  args: sampleArgs,
};

export const Empty: Story = {
  args: {
    ...sampleArgs,
    pendingSection: { ...sampleArgs.pendingSection, rows: [] },
  },
};
