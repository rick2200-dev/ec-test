import type { Meta, StoryObj } from "@storybook/react";
import { useState, type ComponentProps } from "react";
import { SidebarPresenter, type SidebarNavItem } from "./Sidebar.presenter";

const meta: Meta<typeof SidebarPresenter> = {
  title: "Seller/Sidebar",
  component: SidebarPresenter,
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
type Story = StoryObj<typeof SidebarPresenter>;

const sampleNav: SidebarNavItem[] = [
  { href: "/", label: "ダッシュボード", icon: "grid", active: true },
  { href: "/products", label: "商品", icon: "package", active: false },
  { href: "/orders", label: "注文", icon: "shopping-cart", active: false },
  { href: "/inventory", label: "在庫", icon: "warehouse", active: false },
  { href: "/sales", label: "売上", icon: "trending-up", active: false },
  { href: "/subscription", label: "サブスク", icon: "credit-card", active: false },
  { href: "/settings", label: "設定", icon: "settings", active: false },
];

export const Default: Story = {
  args: {
    title: "Seller Console",
    navItems: sampleNav,
    collapsed: false,
    onToggleCollapsed: () => {},
    navAriaLabel: "Main navigation",
    expandAriaLabel: "Expand sidebar",
    collapseAriaLabel: "Collapse sidebar",
    footerLabel: "EC Marketplace v0.1",
  },
};

export const Collapsed: Story = {
  args: {
    ...Default.args,
    collapsed: true,
  },
};

export const ProductsActive: Story = {
  args: {
    ...Default.args,
    navItems: sampleNav.map((item) => ({ ...item, active: item.href === "/products" })),
  },
};

function InteractiveSidebar(args: ComponentProps<typeof SidebarPresenter>) {
  const [collapsed, setCollapsed] = useState(false);
  return (
    <SidebarPresenter
      {...args}
      collapsed={collapsed}
      onToggleCollapsed={() => setCollapsed(!collapsed)}
    />
  );
}

export const Interactive: Story = {
  render: (args) => <InteractiveSidebar {...args} />,
  args: { ...Default.args },
};
