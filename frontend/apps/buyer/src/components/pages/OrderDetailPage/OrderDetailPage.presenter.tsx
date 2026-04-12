import type { ReactNode } from "react";
import Link from "next/link";
import {
  OrderLineItemPresenter,
  type OrderLineItemPresenterProps,
} from "@/components/OrderLineItem";

export interface OrderDetailLineItem extends OrderLineItemPresenterProps {
  id: string;
}

/**
 * Snapshot of the current cancellation request state passed into the
 * presenter. Either a banner is shown (when a request exists) or a
 * cancel button is (when the order is still cancellable) — mutually
 * exclusive states computed by the container.
 */
export interface OrderDetailCancellationSection {
  /** Shown as a colored status banner when there's a request on file. */
  banner?: {
    label: string;
    note?: string;
    tone: "pending" | "approved" | "rejected" | "failed";
  };
  /** Rendered inline under the totals when the order is cancellable. */
  cancelButton?: ReactNode;
}

export interface OrderDetailPagePresenterProps {
  /** Page heading like "注文詳細". */
  title: string;
  /** Back-to-list link label and target. */
  backLabel: string;
  backHref: string;
  /** Order metadata block. */
  orderNumberLabel: string;
  orderedAtLabel: string;
  orderedAtValue: string;
  sellerLabel: string;
  sellerName: string;
  statusLabel: string;
  /** Totals block labels + values. */
  itemsLabel: string;
  subtotalLabel: string;
  subtotalValue: string;
  shippingFeeLabel: string;
  shippingFeeValue: string;
  totalLabel: string;
  totalValue: string;
  /** Optional banner shown when buyer cannot yet contact the seller
   *  (e.g. order still `pending`). Hidden when undefined. */
  purchaseRequiredNotice?: string;
  /** Enriched order lines. */
  lines: OrderDetailLineItem[];
  /** Cancellation banner + button slot. */
  cancellation?: OrderDetailCancellationSection;
}

/**
 * OrderDetailPagePresenter renders /orders/[id]. It is a pure presentational
 * component: the container is responsible for enrichment (image, deleted
 * flag) and for formatting every price / date via the locale-aware helpers.
 */
export function OrderDetailPagePresenter({
  title,
  backLabel,
  backHref,
  orderNumberLabel,
  orderedAtLabel,
  orderedAtValue,
  sellerLabel,
  sellerName,
  statusLabel,
  itemsLabel,
  subtotalLabel,
  subtotalValue,
  shippingFeeLabel,
  shippingFeeValue,
  totalLabel,
  totalValue,
  purchaseRequiredNotice,
  lines,
  cancellation,
}: OrderDetailPagePresenterProps) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      <Link href={backHref} className="text-sm text-blue-600 hover:text-blue-800">
        &larr; {backLabel}
      </Link>
      <h1 className="mt-2 text-2xl font-bold text-gray-900">{title}</h1>

      {/* Order metadata card */}
      <section className="mt-6 rounded-lg border border-gray-200 bg-white p-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-xs text-gray-500">{orderNumberLabel}</p>
            <p className="mt-1 text-sm text-gray-500">
              {orderedAtLabel}: {orderedAtValue}
            </p>
            <p className="mt-1 text-sm font-medium text-gray-900">
              {sellerLabel}: {sellerName}
            </p>
          </div>
          <span className="inline-flex items-center rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700">
            {statusLabel}
          </span>
        </div>
      </section>

      {purchaseRequiredNotice && (
        <div
          className="mt-4 rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800"
          role="note"
        >
          {purchaseRequiredNotice}
        </div>
      )}

      {cancellation?.banner && (
        <div
          className={`mt-4 rounded-md border px-4 py-3 text-sm ${cancellationBannerClasses(cancellation.banner.tone)}`}
          role="note"
        >
          <p className="font-semibold">{cancellation.banner.label}</p>
          {cancellation.banner.note && (
            <p className="mt-1 text-xs">{cancellation.banner.note}</p>
          )}
        </div>
      )}

      {/* Line items */}
      <section className="mt-6 rounded-lg border border-gray-200 bg-white px-6">
        <h2 className="py-4 text-sm font-semibold text-gray-900 border-b border-gray-200">
          {itemsLabel}
        </h2>
        <ul className="divide-y divide-gray-200">
          {lines.map(({ id, ...lineProps }) => (
            <OrderLineItemPresenter key={id} {...lineProps} />
          ))}
        </ul>
      </section>

      {/* Totals */}
      <section className="mt-6 rounded-lg border border-gray-200 bg-white p-6">
        <dl className="space-y-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-gray-600">{subtotalLabel}</dt>
            <dd className="text-gray-900">{subtotalValue}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-gray-600">{shippingFeeLabel}</dt>
            <dd className="text-gray-900">{shippingFeeValue}</dd>
          </div>
          <div className="flex justify-between border-t border-gray-200 pt-2">
            <dt className="text-base font-semibold text-gray-900">{totalLabel}</dt>
            <dd className="text-base font-bold text-gray-900">{totalValue}</dd>
          </div>
        </dl>
      </section>

      {cancellation?.cancelButton && (
        <div className="mt-6 flex justify-end">{cancellation.cancelButton}</div>
      )}
    </div>
  );
}

function cancellationBannerClasses(
  tone: "pending" | "approved" | "rejected" | "failed",
): string {
  switch (tone) {
    case "pending":
      return "border-amber-200 bg-amber-50 text-amber-800";
    case "approved":
      return "border-emerald-200 bg-emerald-50 text-emerald-800";
    case "rejected":
      return "border-slate-200 bg-slate-50 text-slate-700";
    case "failed":
      return "border-red-200 bg-red-50 text-red-800";
  }
}
