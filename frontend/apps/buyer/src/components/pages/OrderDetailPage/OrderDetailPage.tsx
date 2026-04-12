import { notFound } from "next/navigation";
import { getTranslations, getLocale } from "next-intl/server";
import { formatPrice } from "@/lib/mock-data";
import { fetchAPI, getOrderCancellationRequest } from "@/lib/api";
import StartInquiryButton from "@/components/StartInquiryButton";
import CancelOrderButton from "@/components/CancelOrderButton";
import type { OrderDetail, OrderStatus } from "@/lib/types";
import type {
  CancellationRequest,
  CancellationRequestStatus,
} from "@ec-marketplace/types";
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
 * Whether a "cancel this order" button should be shown to the buyer.
 * Mirrors `canOrderBeCancelled` in
 * `backend/services/order/internal/cancellation/domain.go`, and also
 * gates on the current cancellation-request state: a pending or
 * approved request blocks creating a new one (the partial unique index
 * would reject pending dupes anyway, and an approved one means the
 * order is already on its way to cancelled).
 */
function canRequestCancellation(
  status: OrderStatus,
  existing: CancellationRequest | null,
): boolean {
  const statusAllows =
    status === "pending" || status === "paid" || status === "processing";
  if (!statusAllows) return false;
  if (!existing) return true;
  // rejected / failed terminal states leave the door open for a retry.
  return existing.status === "rejected" || existing.status === "failed";
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

  // Fetch the latest cancellation request for this order. 404 → null so
  // we can render the "cancel" button; any other error degrades to null
  // (the container doesn't want to fail the whole page on a cancel-req
  // lookup flake).
  let cancellationRequest: CancellationRequest | null = null;
  try {
    cancellationRequest = await getOrderCancellationRequest(detail.id);
  } catch {
    cancellationRequest = null;
  }
  const cancellable = canRequestCancellation(detail.status, cancellationRequest);
  const cancellationSection = buildCancellationSection(
    cancellationRequest,
    cancellable,
    detail.id,
    t,
  );

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
      cancellation={cancellationSection}
    />
  );
}

/**
 * Build the cancellation section prop for the presenter. Mutually
 * exclusive: if a request is on file we show its banner; otherwise
 * (and if the order is still cancellable) we show the button.
 */
function buildCancellationSection(
  request: CancellationRequest | null,
  cancellable: boolean,
  orderId: string,
  t: Awaited<ReturnType<typeof getTranslations>>,
): {
  banner?: {
    label: string;
    note?: string;
    tone: "pending" | "approved" | "rejected" | "failed";
  };
  cancelButton?: React.ReactNode;
} | undefined {
  const section: {
    banner?: {
      label: string;
      note?: string;
      tone: CancellationRequestStatus;
    };
    cancelButton?: React.ReactNode;
  } = {};

  if (request) {
    const status = request.status;
    section.banner = {
      label: t(`orders.cancellation.statusBadge.${status}`),
      note: cancellationNote(status, request, t),
      tone: status,
    };
  }
  if (cancellable) {
    section.cancelButton = <CancelOrderButton orderId={orderId} />;
  }
  if (!section.banner && !section.cancelButton) {
    return undefined;
  }
  return section;
}

function cancellationNote(
  status: CancellationRequestStatus,
  request: CancellationRequest,
  t: Awaited<ReturnType<typeof getTranslations>>,
): string | undefined {
  switch (status) {
    case "pending":
      return t("orders.cancellation.statusNote.pending");
    case "approved":
      return t("orders.cancellation.statusNote.approved");
    case "rejected":
      return t("orders.cancellation.statusNote.rejected", {
        comment: request.seller_comment ?? "",
      });
    case "failed":
      return t("orders.cancellation.statusNote.failed");
    default:
      return undefined;
  }
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
