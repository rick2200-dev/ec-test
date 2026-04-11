import { notFound } from "next/navigation";
import { getTranslations, getLocale } from "next-intl/server";
import { formatPrice } from "@/lib/mock-data";
import { fetchAPI } from "@/lib/api";
import StartInquiryButton from "@/components/StartInquiryButton";
import type { OrderDetail, OrderStatus } from "@/lib/types";
import {
  OrderDetailPagePresenter,
  type OrderDetailLineItem,
} from "./OrderDetailPage.presenter";

interface OrderDetailPageProps {
  orderId: string;
}

/**
 * Order status at which a buyer → seller inquiry is allowed. Mirrors the
 * backend gate (inquiry-svc rejects unpaid orders with 403); this is only a
 * UX hint so we hide the CTA when we already know it would fail.
 */
function canContactSeller(status: OrderStatus): boolean {
  return (
    status === "paid" ||
    status === "processing" ||
    status === "shipped" ||
    status === "delivered" ||
    status === "completed"
  );
}

/**
 * Server component wrapper for the /orders/[id] route. Fetches the enriched
 * order detail from the gateway (which already layered catalog image_url /
 * product_slug / is_deleted on each line) and hands it to the presenter.
 */
export async function OrderDetailPage({ orderId }: OrderDetailPageProps) {
  const t = await getTranslations();
  const locale = await getLocale();

  const res = await fetchAPI(`/api/v1/buyer/orders/${encodeURIComponent(orderId)}`, {
    cache: "no-store",
  });
  if (res.status === 404) {
    notFound();
  }
  if (!res.ok) {
    // Bubble up as 404 rather than crashing — misroutes and upstream
    // hiccups shouldn't pop a 500 in the buyer app.
    notFound();
  }
  const detail = (await res.json()) as OrderDetail;

  const currency = detail.currency;
  const dateFormatter = new Intl.DateTimeFormat(locale === "ja" ? "ja-JP" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

  const labels = {
    quantity: t("orders.quantityLabel"),
    deletedBadge: t("orders.lineDeletedBadge"),
    imageMissing: t("orders.lineImageMissing"),
  };

  const contactable = canContactSeller(detail.status);

  const lines: OrderDetailLineItem[] = detail.lines.map((l) => ({
    id: l.id,
    productName: l.product_name,
    skuCode: l.sku_code,
    quantity: l.quantity,
    unitPriceLabel: formatPrice(l.unit_price, currency),
    lineTotalLabel: formatPrice(l.line_total, currency),
    imageUrl: l.image_url,
    productHref: l.is_deleted || !l.product_slug ? "" : `/products/${l.product_slug}`,
    isDeleted: l.is_deleted,
    labels,
    // Offer the inquiry CTA per line, but skip lines whose referenced SKU
    // has been archived — the inquiry POST would fail on a missing sku
    // anyway, so don't dangle a broken button.
    actionSlot:
      contactable && !l.is_deleted ? (
        <StartInquiryButton
          sellerId={detail.seller_id}
          skuId={l.sku_id}
          productName={l.product_name}
          skuCode={l.sku_code}
        />
      ) : undefined,
  }));

  return (
    <OrderDetailPagePresenter
      title={t("orders.detailTitle")}
      backLabel={t("orders.backToList")}
      backHref="/orders"
      orderNumberLabel={t("orders.orderNumber", { id: detail.id })}
      orderedAtLabel={t("orders.orderedAt")}
      orderedAtValue={dateFormatter.format(new Date(detail.created_at))}
      sellerLabel={t("orders.sellerLabel")}
      sellerName={detail.seller_name || t("orders.unknownSeller")}
      statusLabel={statusLabel(detail.status, t)}
      itemsLabel={t("orders.itemsLabel")}
      subtotalLabel={t("orders.subtotalLabel")}
      subtotalValue={formatPrice(detail.subtotal_amount, currency)}
      shippingFeeLabel={t("orders.shippingFeeLabel")}
      shippingFeeValue={formatPrice(detail.shipping_fee, currency)}
      totalLabel={t("orders.totalLabel")}
      totalValue={formatPrice(detail.total_amount, currency)}
      purchaseRequiredNotice={contactable ? undefined : t("orders.purchaseRequiredNotice")}
      lines={lines}
    />
  );
}

function statusLabel(
  status: OrderStatus,
  t: Awaited<ReturnType<typeof getTranslations>>,
): string {
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
