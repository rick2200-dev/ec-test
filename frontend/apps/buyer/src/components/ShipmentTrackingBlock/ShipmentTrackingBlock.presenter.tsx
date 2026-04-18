import type { ShipmentStatus } from "@ec-marketplace/types";

export interface ShipmentTrackingBlockPresenterProps {
  /** Null = no shipment registered yet (show "preparing"). */
  status: ShipmentStatus | null;
  carrier?: string;
  trackingNumber?: string;
  shippedAt?: string;
  deliveredAt?: string;
  labels: {
    sectionTitle: string;
    preparing: string;
    carrier: string;
    trackingNumber: string;
    shippedAt: string;
    deliveredAt: string;
    statusLabels: Record<ShipmentStatus, string>;
  };
  locale: string;
}

const STATUS_TONE: Record<ShipmentStatus, string> = {
  pending: "bg-gray-100 text-gray-700",
  ready_to_ship: "bg-yellow-100 text-yellow-800",
  shipped: "bg-blue-100 text-blue-800",
  delivered: "bg-green-100 text-green-800",
  cancelled: "bg-red-100 text-red-700",
};

export function ShipmentTrackingBlockPresenter({
  status,
  carrier,
  trackingNumber,
  shippedAt,
  deliveredAt,
  labels,
  locale,
}: ShipmentTrackingBlockPresenterProps) {
  const dtf = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" });

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-5">
      <h2 className="text-base font-semibold text-gray-900">{labels.sectionTitle}</h2>
      {status === null ? (
        <p className="mt-2 text-sm text-gray-500">{labels.preparing}</p>
      ) : (
        <div className="mt-3 space-y-2 text-sm">
          <span
            className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_TONE[status]}`}
          >
            {labels.statusLabels[status]}
          </span>
          {carrier && (
            <p className="flex justify-between">
              <span className="text-gray-500">{labels.carrier}</span>
              <span className="text-gray-900">{carrier}</span>
            </p>
          )}
          {trackingNumber && (
            <p className="flex justify-between">
              <span className="text-gray-500">{labels.trackingNumber}</span>
              <span className="font-mono text-gray-900">{trackingNumber}</span>
            </p>
          )}
          {shippedAt && (
            <p className="flex justify-between">
              <span className="text-gray-500">{labels.shippedAt}</span>
              <span className="text-gray-900">{dtf.format(new Date(shippedAt))}</span>
            </p>
          )}
          {deliveredAt && (
            <p className="flex justify-between">
              <span className="text-gray-500">{labels.deliveredAt}</span>
              <span className="text-gray-900">{dtf.format(new Date(deliveredAt))}</span>
            </p>
          )}
        </div>
      )}
    </section>
  );
}
