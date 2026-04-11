import type { ReactNode } from "react";
import Link from "next/link";

export interface OrderDetailLineRow {
  key: string;
  productName: string;
  skuCode: string;
  sellerName: string;
  quantity: number;
  unitPriceLabel: string;
  subtotalLabel: string;
  /** Button / trigger rendered next to the line (StartInquiryButton or disabled placeholder). */
  actionSlot?: ReactNode;
}

export interface OrderDetailPagePresenterProps {
  backHref: string;
  backLabel: string;
  orderId: string;
  orderIdLabel: string;
  orderedAt: string;
  orderedAtLabel: string;
  statusLabel: string;
  totalLabel: string;
  totalValue: string;
  productsLabel: string;
  purchaseRequiredNotice?: string;
  lines: OrderDetailLineRow[];
}

export function OrderDetailPagePresenter({
  backHref,
  backLabel,
  orderId,
  orderIdLabel,
  orderedAt,
  orderedAtLabel,
  statusLabel,
  totalLabel,
  totalValue,
  productsLabel,
  purchaseRequiredNotice,
  lines,
}: OrderDetailPagePresenterProps) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8 space-y-5">
      <Link
        href={backHref}
        className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
      >
        ← {backLabel}
      </Link>

      <header className="rounded-lg border border-gray-200 bg-white p-4">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-4">
          <div>
            <dt className="text-xs uppercase text-gray-500">{orderIdLabel}</dt>
            <dd className="mt-1 font-mono text-xs text-gray-800">{orderId}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-gray-500">{orderedAtLabel}</dt>
            <dd className="mt-1 text-sm text-gray-800">{orderedAt}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-gray-500">ステータス</dt>
            <dd className="mt-1 text-sm text-gray-800">{statusLabel}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase text-gray-500">{totalLabel}</dt>
            <dd className="mt-1 text-sm font-semibold text-gray-900">{totalValue}</dd>
          </div>
        </dl>
      </header>

      {purchaseRequiredNotice && (
        <div
          className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800"
          role="note"
        >
          {purchaseRequiredNotice}
        </div>
      )}

      <section className="rounded-lg border border-gray-200 bg-white shadow-sm">
        <h2 className="border-b border-gray-200 px-4 py-3 text-sm font-semibold text-gray-900">
          {productsLabel}
        </h2>
        <ul className="divide-y divide-gray-200">
          {lines.map((line) => (
            <li
              key={line.key}
              className="flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <div className="text-sm font-medium text-gray-900">{line.productName}</div>
                <div className="mt-0.5 text-xs text-gray-500">
                  <span className="font-mono">{line.skuCode}</span>
                  <span className="mx-2">·</span>
                  <span>{line.sellerName}</span>
                </div>
                <div className="mt-1 text-xs text-gray-600">
                  {line.unitPriceLabel} × {line.quantity} = {line.subtotalLabel}
                </div>
              </div>
              {line.actionSlot && <div className="shrink-0">{line.actionSlot}</div>}
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
