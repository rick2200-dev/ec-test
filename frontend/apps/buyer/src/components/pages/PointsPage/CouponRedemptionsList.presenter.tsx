import { formatCurrency } from "@ec-marketplace/ui-utils";
import type { CouponRedemption } from "@ec-marketplace/types";

export interface CouponRedemptionsListPresenterProps {
  redemptions: CouponRedemption[];
  emptyLabel: string;
  loadingLabel: string;
  loaded: boolean;
  columns: {
    date: string;
    orderId: string;
    discount: string;
    refunded: string;
    status: string;
  };
  statusLabels: {
    active: string;
    refundedFully: string;
    refundedPartially: string;
  };
  locale: string;
}

export function CouponRedemptionsListPresenter({
  redemptions,
  emptyLabel,
  loadingLabel,
  loaded,
  columns,
  statusLabels,
  locale,
}: CouponRedemptionsListPresenterProps) {
  if (!loaded) {
    return <p className="py-8 text-center text-sm text-gray-500">{loadingLabel}</p>;
  }
  if (redemptions.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-gray-300 py-8 text-center text-sm text-gray-500">
        {emptyLabel}
      </p>
    );
  }
  const dtf = new Intl.DateTimeFormat(locale, { dateStyle: "medium" });

  const statusFor = (r: CouponRedemption): string => {
    if (r.refunded_at) return statusLabels.refundedFully;
    if (r.refunded_amount > 0) return statusLabels.refundedPartially;
    return statusLabels.active;
  };

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          <tr>
            <th scope="col" className="px-4 py-3">
              {columns.date}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.orderId}
            </th>
            <th scope="col" className="px-4 py-3 text-right">
              {columns.discount}
            </th>
            <th scope="col" className="px-4 py-3 text-right">
              {columns.refunded}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.status}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {redemptions.map((r) => (
            <tr key={r.id}>
              <td className="px-4 py-3 text-gray-700">{dtf.format(new Date(r.committed_at))}</td>
              <td className="px-4 py-3 font-mono text-xs text-gray-600">{r.order_id}</td>
              <td className="px-4 py-3 text-right text-gray-900">
                −{formatCurrency(r.discount_applied)}
              </td>
              <td className="px-4 py-3 text-right text-gray-700">
                {r.refunded_amount > 0 ? formatCurrency(r.refunded_amount) : "—"}
              </td>
              <td className="px-4 py-3 text-gray-700">{statusFor(r)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
