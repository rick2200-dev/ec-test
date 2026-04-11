"use client";

import { useMemo, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import {
  NotificationsPagePresenter,
  type NotificationCardItem,
  type NotificationCategory,
  type NotificationFilter,
} from "./NotificationsPage.presenter";

/**
 * Seller notifications feed. The backend notification service is currently
 * email-only and does not persist notifications, so this page renders an
 * in-memory mock list. Swap `mockNotifications` for a real API call when a
 * `/api/v1/seller/notifications` endpoint ships.
 */
type MockNotificationType =
  | "orderCreated"
  | "orderPaid"
  | "inventoryLowStock"
  | "inquiryReceived"
  | "sellerApproved";

interface MockNotification {
  id: string;
  type: MockNotificationType;
  createdAt: string;
  unread: boolean;
  href?: string;
  vars: Record<string, string>;
}

const CATEGORY_BY_TYPE: Record<MockNotificationType, NotificationCategory> = {
  orderCreated: "order",
  orderPaid: "order",
  inventoryLowStock: "inventory",
  inquiryReceived: "inquiry",
  sellerApproved: "account",
};

const mockNotifications: MockNotification[] = [
  {
    id: "ntf-s-1",
    type: "orderCreated",
    createdAt: "2026-04-12T08:44:00Z",
    unread: true,
    href: "/orders",
    vars: { orderId: "ord-42", amount: "¥14,080" },
  },
  {
    id: "ntf-s-2",
    type: "inventoryLowStock",
    createdAt: "2026-04-11T21:05:00Z",
    unread: true,
    href: "/inventory",
    vars: { sku: "SKU-RED-M", productName: "Premium T-Shirt", quantity: "3" },
  },
  {
    id: "ntf-s-3",
    type: "inquiryReceived",
    createdAt: "2026-04-11T13:18:00Z",
    unread: true,
    href: "/inquiries",
    vars: { subject: "発送時期について" },
  },
  {
    id: "ntf-s-4",
    type: "orderPaid",
    createdAt: "2026-04-10T09:30:00Z",
    unread: false,
    href: "/orders",
    vars: { orderId: "ord-41" },
  },
  {
    id: "ntf-s-5",
    type: "sellerApproved",
    createdAt: "2026-03-28T12:00:00Z",
    unread: false,
    vars: {},
  },
];

export default function NotificationsPage() {
  const t = useTranslations("notifications");
  const locale = useLocale();
  const [filter, setFilter] = useState<NotificationFilter>("all");
  const [readOverrides, setReadOverrides] = useState<Set<string>>(new Set());
  const [markedAllRead, setMarkedAllRead] = useState(false);

  const dateFormatter = useMemo(
    () =>
      new Intl.DateTimeFormat(locale === "ja" ? "ja-JP" : "en-US", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      }),
    [locale],
  );

  const cards: NotificationCardItem[] = useMemo(() => {
    const allCards = mockNotifications.map<NotificationCardItem>((n) => ({
      id: n.id,
      createdAtLabel: dateFormatter.format(new Date(n.createdAt)),
      title: titleForType(n.type, t),
      body: bodyForType(n.type, n.vars, t),
      category: CATEGORY_BY_TYPE[n.type],
      unread: n.unread && !markedAllRead && !readOverrides.has(n.id),
      href: n.href,
    }));

    return filter === "unread" ? allCards.filter((c) => c.unread) : allCards;
  }, [dateFormatter, filter, markedAllRead, readOverrides, t]);

  return (
    <NotificationsPagePresenter
      title={t("title")}
      description={t("description")}
      emptyMessage={t("empty")}
      unreadBadgeLabel={t("unreadBadge")}
      markAllReadLabel={t("markAllRead")}
      filterAllLabel={t("filter.all")}
      filterUnreadLabel={t("filter.unread")}
      filter={filter}
      onFilterChange={setFilter}
      onMarkAllRead={() => {
        setMarkedAllRead(true);
        setReadOverrides(new Set(mockNotifications.map((n) => n.id)));
      }}
      notifications={cards}
    />
  );
}

type T = ReturnType<typeof useTranslations<"notifications">>;

function titleForType(type: MockNotificationType, t: T): string {
  switch (type) {
    case "orderCreated":
      return t("type.orderCreated");
    case "orderPaid":
      return t("type.orderPaid");
    case "inventoryLowStock":
      return t("type.inventoryLowStock");
    case "inquiryReceived":
      return t("type.inquiryReceived");
    case "sellerApproved":
      return t("type.sellerApproved");
  }
}

function bodyForType(
  type: MockNotificationType,
  vars: Record<string, string>,
  t: T,
): string {
  switch (type) {
    case "orderCreated":
      return t("body.orderCreated", {
        orderId: vars.orderId ?? "",
        amount: vars.amount ?? "",
      });
    case "orderPaid":
      return t("body.orderPaid", { orderId: vars.orderId ?? "" });
    case "inventoryLowStock":
      return t("body.inventoryLowStock", {
        sku: vars.sku ?? "",
        productName: vars.productName ?? "",
        quantity: vars.quantity ?? "",
      });
    case "inquiryReceived":
      return t("body.inquiryReceived", { subject: vars.subject ?? "" });
    case "sellerApproved":
      return t("body.sellerApproved");
  }
}
