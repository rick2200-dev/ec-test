import type { Shipment, ShipmentStatus } from "@ec-marketplace/types";

export interface ShipmentsListPresenterProps {
  shipments: Shipment[];
  emptyLabel: string;
  loadingLabel: string;
  loaded: boolean;
  statusLabels: Record<ShipmentStatus, string>;
  columns: {
    orderId: string;
    status: string;
    carrier: string;
    tracking: string;
    shippedAt: string;
    actions: string;
  };
  /** Per-row action buttons (register / mark delivered) provided by the container. */
  renderActions: (s: Shipment) => React.ReactNode;
  locale: string;
}

const STATUS_BADGE_COLOR: Record<ShipmentStatus, string> = {
  pending: "bg-gray-100 text-gray-700",
  ready_to_ship: "bg-yellow-100 text-yellow-800",
  shipped: "bg-blue-100 text-blue-800",
  delivered: "bg-green-100 text-green-800",
  cancelled: "bg-red-100 text-red-700",
};

export function ShipmentsListPresenter({
  shipments,
  emptyLabel,
  loadingLabel,
  loaded,
  statusLabels,
  columns,
  renderActions,
  locale,
}: ShipmentsListPresenterProps) {
  if (!loaded) {
    return <p className="py-8 text-center text-sm text-text-secondary">{loadingLabel}</p>;
  }
  if (shipments.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-border py-8 text-center text-sm text-text-secondary">
        {emptyLabel}
      </p>
    );
  }
  const dtf = new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short" });

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-white shadow-sm">
      <table className="min-w-full divide-y divide-border text-sm">
        <thead className="bg-surface text-left text-xs font-medium uppercase tracking-wider text-text-secondary">
          <tr>
            <th scope="col" className="px-4 py-3">
              {columns.orderId}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.status}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.carrier}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.tracking}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.shippedAt}
            </th>
            <th scope="col" className="px-4 py-3 text-right">
              {columns.actions}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border bg-white">
          {shipments.map((s) => (
            <tr key={s.id}>
              <td className="px-4 py-3 font-mono text-xs text-text-secondary">
                {s.order_id.slice(0, 8)}
              </td>
              <td className="px-4 py-3">
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_BADGE_COLOR[s.status]}`}
                >
                  {statusLabels[s.status] ?? s.status}
                </span>
              </td>
              <td className="px-4 py-3 text-text-primary">{s.carrier ?? "—"}</td>
              <td className="px-4 py-3 font-mono text-xs text-text-primary">
                {s.tracking_number ?? "—"}
              </td>
              <td className="px-4 py-3 text-text-secondary">
                {s.shipped_at ? dtf.format(new Date(s.shipped_at)) : "—"}
              </td>
              <td className="px-4 py-3 text-right">{renderActions(s)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
