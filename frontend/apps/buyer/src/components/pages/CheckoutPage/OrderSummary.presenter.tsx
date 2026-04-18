import { formatCurrency } from "@ec-marketplace/ui-utils";

export interface OrderSummaryPresenterProps {
  itemsCount: number;
  subtotal: number;
  couponDiscount: number;
  pointsRedeemed: number;
  total: number;
  labels: {
    title: string;
    items: string;
    subtotal: string;
    couponDiscount: string;
    pointsRedeemed: string;
    total: string;
  };
}

export function OrderSummaryPresenter({
  itemsCount,
  subtotal,
  couponDiscount,
  pointsRedeemed,
  total,
  labels,
}: OrderSummaryPresenterProps) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-6">
      <h2 className="text-base font-semibold text-gray-900">{labels.title}</h2>
      <dl className="mt-4 space-y-2 text-sm text-gray-700">
        <div className="flex justify-between">
          <dt>{labels.items}</dt>
          <dd>{itemsCount}</dd>
        </div>
        <div className="flex justify-between">
          <dt>{labels.subtotal}</dt>
          <dd>{formatCurrency(subtotal)}</dd>
        </div>
        {couponDiscount > 0 && (
          <div className="flex justify-between text-green-700">
            <dt>{labels.couponDiscount}</dt>
            <dd>−{formatCurrency(couponDiscount)}</dd>
          </div>
        )}
        {pointsRedeemed > 0 && (
          <div className="flex justify-between text-green-700">
            <dt>{labels.pointsRedeemed}</dt>
            <dd>−{formatCurrency(pointsRedeemed)}</dd>
          </div>
        )}
        <div className="flex justify-between border-t border-gray-200 pt-3 text-base font-bold text-gray-900">
          <dt>{labels.total}</dt>
          <dd>{formatCurrency(total)}</dd>
        </div>
      </dl>
    </section>
  );
}
