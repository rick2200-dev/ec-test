"use client";

import { usePathname } from "next/navigation";
import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  SidebarPresenter,
  type SidebarIconKey,
  type SidebarNavItem,
} from "./Sidebar.presenter";

interface NavItemDef {
  href: string;
  labelKey: string;
  icon: SidebarIconKey;
}

const navItemDefs: NavItemDef[] = [
  { href: "/", labelKey: "sidebar.dashboard", icon: "grid" },
  { href: "/products", labelKey: "sidebar.products", icon: "package" },
  { href: "/orders", labelKey: "sidebar.orders", icon: "shopping-cart" },
  { href: "/inventory", labelKey: "sidebar.inventory", icon: "warehouse" },
  { href: "/sales", labelKey: "sidebar.sales", icon: "trending-up" },
  { href: "/subscription", labelKey: "sidebar.subscription", icon: "credit-card" },
  { href: "/settings", labelKey: "sidebar.settings", icon: "settings" },
];

export default function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const t = useTranslations();

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/";
    return pathname.startsWith(href);
  };

  const navItems: SidebarNavItem[] = navItemDefs.map((def) => ({
    href: def.href,
    label: t(def.labelKey),
    icon: def.icon,
    active: isActive(def.href),
  }));

  return (
    <SidebarPresenter
      title={t("sidebar.title")}
      navItems={navItems}
      collapsed={collapsed}
      onToggleCollapsed={() => setCollapsed(!collapsed)}
      navAriaLabel={t("a11y.mainNav")}
      expandAriaLabel={t("a11y.expandSidebar")}
      collapseAriaLabel={t("a11y.collapseSidebar")}
      footerLabel="EC Marketplace v0.1"
    />
  );
}
