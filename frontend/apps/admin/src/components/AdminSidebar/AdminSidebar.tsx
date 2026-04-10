"use client";

import { usePathname } from "next/navigation";
import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  AdminSidebarPresenter,
  type AdminSidebarIconKey,
  type AdminSidebarNavItem,
} from "./AdminSidebar.presenter";

interface NavItemDef {
  href: string;
  labelKey: string;
  icon: AdminSidebarIconKey;
}

const navItemDefs: NavItemDef[] = [
  { href: "/", labelKey: "sidebar.dashboard", icon: "grid" },
  { href: "/tenants", labelKey: "sidebar.tenants", icon: "building" },
  { href: "/sellers", labelKey: "sidebar.sellers", icon: "users" },
  { href: "/plans", labelKey: "sidebar.plans", icon: "credit-card" },
  { href: "/commissions", labelKey: "sidebar.commissions", icon: "percent" },
  { href: "/analytics", labelKey: "sidebar.analytics", icon: "chart" },
  { href: "/settings", labelKey: "sidebar.settings", icon: "settings" },
];

export default function AdminSidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const t = useTranslations();

  const isActive = (href: string) => {
    if (href === "/") return pathname === "/";
    return pathname.startsWith(href);
  };

  const navItems: AdminSidebarNavItem[] = navItemDefs.map((def) => ({
    href: def.href,
    label: t(def.labelKey),
    icon: def.icon,
    active: isActive(def.href),
  }));

  return (
    <AdminSidebarPresenter
      title={t("sidebar.title")}
      navItems={navItems}
      collapsed={collapsed}
      onToggleCollapsed={() => setCollapsed(!collapsed)}
      navAriaLabel={t("a11y.mainNav")}
      expandAriaLabel={t("a11y.expandSidebar")}
      collapseAriaLabel={t("a11y.collapseSidebar")}
      versionLabel={t("sidebar.version")}
    />
  );
}
