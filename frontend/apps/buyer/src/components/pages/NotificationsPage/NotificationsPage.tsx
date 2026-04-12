import { getTranslations, getLocale } from "next-intl/server";
import {
  NotificationsPagePresenter,
  type NotificationCardItem,
} from "./NotificationsPage.presenter";

/**
 * Buyer notification feed. Backend currently has no persisted notifications
 * (the notification service is an email-only pub/sub worker), so this screen
 * renders a hard-coded mock list for now. Replace `mockNotifications` with a
 * real `fetchAPI("/api/v1/buyer/notifications")` call once the endpoint ships.
 */
type MockNotificationType = "orderPaid" | "orderShipped" | "orderDelivered" | "inquiryReplied";

interface MockNotification {
  id: string;
  type: MockNotificationType;
  createdAt: string;
  unread: boolean;
  href?: string;
  /** Per-type template variables (mirrors the i18n message placeholders). */
  vars: Record<string, string>;
}

const mockNotifications: MockNotification[] = [
  {
    id: "ntf-b-1",
    type: "orderShipped",
    createdAt: "2026-04-11T09:12:00Z",
    unread: true,
    href: "/orders/ord-1",
    vars: { orderId: "ord-1", tracking: "JP1234567890" },
  },
  {
    id: "ntf-b-2",
    type: "inquiryReplied",
    createdAt: "2026-04-10T15:40:00Z",
    unread: true,
    href: "/inquiries",
    vars: { subject: "発送時期について" },
  },
  {
    id: "ntf-b-3",
    type: "orderPaid",
    createdAt: "2026-04-09T11:05:00Z",
    unread: false,
    href: "/orders/ord-2",
    vars: { orderId: "ord-2" },
  },
  {
    id: "ntf-b-4",
    type: "orderDelivered",
    createdAt: "2026-04-03T18:22:00Z",
    unread: false,
    href: "/orders/ord-3",
    vars: { orderId: "ord-3" },
  },
];

export async function NotificationsPage() {
  const t = await getTranslations();
  const locale = await getLocale();

  const dateFormatter = new Intl.DateTimeFormat(locale === "ja" ? "ja-JP" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

  const cards: NotificationCardItem[] = mockNotifications.map((n) => ({
    id: n.id,
    createdAtLabel: dateFormatter.format(new Date(n.createdAt)),
    title: titleForType(n.type, t),
    body: bodyForType(n.type, n.vars, t),
    href: n.href,
    unread: n.unread,
  }));

  return (
    <NotificationsPagePresenter
      title={t("notifications.pageTitle")}
      description={t("notifications.description")}
      emptyMessage={t("notifications.empty")}
      unreadBadgeLabel={t("notifications.unreadBadge")}
      notifications={cards}
    />
  );
}

type T = Awaited<ReturnType<typeof getTranslations>>;

function titleForType(type: MockNotificationType, t: T): string {
  // next-intl types require literal message keys; use a switch so TypeScript
  // verifies each key exists in the message catalogue.
  switch (type) {
    case "orderPaid":
      return t("notifications.type.orderPaid");
    case "orderShipped":
      return t("notifications.type.orderShipped");
    case "orderDelivered":
      return t("notifications.type.orderDelivered");
    case "inquiryReplied":
      return t("notifications.type.inquiryReplied");
  }
}

function bodyForType(type: MockNotificationType, vars: Record<string, string>, t: T): string {
  switch (type) {
    case "orderPaid":
      return t("notifications.body.orderPaid", { orderId: vars.orderId ?? "" });
    case "orderShipped":
      return t("notifications.body.orderShipped", {
        orderId: vars.orderId ?? "",
        tracking: vars.tracking ?? "",
      });
    case "orderDelivered":
      return t("notifications.body.orderDelivered", { orderId: vars.orderId ?? "" });
    case "inquiryReplied":
      return t("notifications.body.inquiryReplied", { subject: vars.subject ?? "" });
  }
}
