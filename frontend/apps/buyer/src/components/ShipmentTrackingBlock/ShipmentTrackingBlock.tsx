import { getTranslations, getLocale } from "next-intl/server";
import type { Shipment, ShipmentStatus } from "@ec-marketplace/types";
import { getOrderShipment } from "@/lib/api";
import { ShipmentTrackingBlockPresenter } from "./ShipmentTrackingBlock.presenter";

/**
 * Server-fetched block that renders the shipment tracking state for an
 * order. Failure (network / 401 / other) degrades to the "preparing"
 * state — the order detail page should not crash on a flaky shipment
 * read.
 */
export default async function ShipmentTrackingBlock({ orderId }: { orderId: string }) {
  const t = await getTranslations("orders.shipment");
  const locale = await getLocale();

  let shipment: Shipment | null = null;
  try {
    shipment = await getOrderShipment(orderId);
  } catch {
    shipment = null;
  }

  const statusLabels: Record<ShipmentStatus, string> = {
    pending: t("status.pending"),
    ready_to_ship: t("status.readyToShip"),
    shipped: t("status.shipped"),
    delivered: t("status.delivered"),
    cancelled: t("status.cancelled"),
  };

  return (
    <ShipmentTrackingBlockPresenter
      status={shipment?.status ?? null}
      carrier={shipment?.carrier ?? undefined}
      trackingNumber={shipment?.tracking_number ?? undefined}
      shippedAt={shipment?.shipped_at ?? undefined}
      deliveredAt={shipment?.delivered_at ?? undefined}
      labels={{
        sectionTitle: t("sectionTitle"),
        preparing: t("preparing"),
        carrier: t("carrier"),
        trackingNumber: t("trackingNumber"),
        shippedAt: t("shippedAt"),
        deliveredAt: t("deliveredAt"),
        statusLabels,
      }}
      locale={locale}
    />
  );
}
