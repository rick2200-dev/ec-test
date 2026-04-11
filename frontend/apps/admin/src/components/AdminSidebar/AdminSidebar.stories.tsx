import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import {
  AdminSidebarPresenter,
  type AdminSidebarNavItem,
  type AdminSidebarPresenterProps,
} from "./AdminSidebar.presenter";

const meta: Meta<typeof AdminSidebarPresenter> = {
  title: "Admin/AdminSidebar",
  component: AdminSidebarPresenter,
  parameters: { layout: "fullscreen" },
  decorators: [
    (Story) => (
      <div style={{ height: "100vh" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof AdminSidebarPresenter>;

const sampleNav: AdminSidebarNavItem[] = [
  { href: "/", label: "ダッシュボード", icon: "grid", active: true },
  { href: "/tenants", label: "テナント", icon: "building", active: false },
  { href: "/sellers", label: "セラー", icon: "users", active: false },
  { href: "/plans", label: "プラン", icon: "credit-card", active: false },
  { href: "/commissions", label: "手数料", icon: "percent", active: false },
  { href: "/analytics", label: "分析", icon: "chart", active: false },
  { href: "/settings", label: "設定", icon: "settings", active: false },
];

export const Default: Story = {
  args: {
    title: "Admin Console",
    navItems: sampleNav,
    collapsed: false,
    onToggleCollapsed: () => {},
    navAriaLabel: "Main navigation",
    expandAriaLabel: "Expand sidebar",
    collapseAriaLabel: "Collapse sidebar",
    versionLabel: "v0.1",
  },
};

export const Collapsed: Story = {
  args: { ...Default.args, collapsed: true },
};

export const SellersActive: Story = {
  args: {
    ...Default.args,
    navItems: sampleNav.map((item) => ({ ...item, active: item.href === "/sellers" })),
  },
};

function InteractiveStory(args: AdminSidebarPresenterProps) {
  const [collapsed, setCollapsed] = useState(false);
  return (
    <AdminSidebarPresenter
      {...args}
      collapsed={collapsed}
      onToggleCollapsed={() => setCollapsed(!collapsed)}
    />
  );
}

export const Interactive: Story = {
  render: (args) => <InteractiveStory {...args} />,
  args: { ...Default.args },
};
