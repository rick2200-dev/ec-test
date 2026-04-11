import { getTranslations, getLocale } from "next-intl/server";
import type { OrderListResponse, OrderSummary, OrderStatus } from "@/lib/types";
import { formatPrice } from "@/lib/mock-data";
import { fetchAPI } from "@/lib/api";
import {
  OrderHistoryPagePresenter,
  type OrderHistoryCardItem,
} from "./OrderHistoryPage.presenter";

/**
 * Server component wrapper around OrderHistoryPagePresenter. Calls the
 * gateway's list endpoint and maps the raw order rows to localized,
 * pre-formatted card data for the presenter.
 *
 * Auth: currently relies on whatever (none, today) the shared fetchAPI
 * layer sends. Wiring JWT forwarding is a follow-up — see plan notes.
 */
export async function OrderHistoryPage() {
  const t = await getTranslations();
  const locale = await getLocale();

  let orders: OrderSummary[] = [];
  try {
    const res = await fetchAPI("/api/v1/buyer/orders?limit=20", {
      cache: "no-store",
    });
    if (res.ok) {
      const body = (await res.json()) as OrderListResponse;
      orders = body.items ?? [];
    }
  } catch {
    // Silently degrade to empty state — logging hook TBD.
  }

  const dateFormatter = new Intl.DateTimeFormat(locale === "ja" ? "ja-JP" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

  const cards: OrderHistoryCardItem[] = orders.map((o) => ({
    id: o.id,
    href: `/orders/${o.id}`,
    createdAtLabel: dateFormatter.format(new Date(o.created_at)),
    sellerName: o.seller_name || t("orders.unknownSeller"),
    statusLabel: statusLabel(o.status, t),
    totalLabel: formatPrice(o.total_amount, o.currency),
  }));

  return (
    <OrderHistoryPagePresenter
      title={t("orders.pageTitle")}
      sellerLabel={t("orders.sellerLabel")}
      totalLabel={t("orders.totalLabel")}
      emptyMessage={t("orders.empty")}
      browseCtaLabel={t("orders.browseCta")}
      browseCtaHref="/products"
      orders={cards}
    />
  );
}

function statusLabel(
  status: OrderStatus,
  t: Awaited<ReturnType<typeof getTranslations>>,
): string {
  // next-intl types are strict about message keys; use a typed map so
  // unknown statuses fall back to the raw value instead of throwing.
  switch (status) {
    case "pending":
      return t("orders.status.pending");
    case "paid":
      return t("orders.status.paid");
    case "processing":
      return t("orders.status.processing");
    case "shipped":
      return t("orders.status.shipped");
    case "delivered":
      return t("orders.status.delivered");
    case "completed":
      return t("orders.status.completed");
    case "cancelled":
      return t("orders.status.cancelled");
    default:
      return status;
  }
}
