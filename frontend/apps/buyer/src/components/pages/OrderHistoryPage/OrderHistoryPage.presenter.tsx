import Link from "next/link";

export interface OrderHistoryCardItem {
  id: string;
  href: string;
  /** Pre-formatted localized date label (e.g. "2026年4月11日"). */
  createdAtLabel: string;
  sellerName: string;
  /** Localized status label (e.g. "発送済み"). */
  statusLabel: string;
  totalLabel: string;
}

export interface OrderHistoryPagePresenterProps {
  title: string;
  sellerLabel: string;
  totalLabel: string;
  emptyMessage: string;
  browseCtaLabel: string;
  browseCtaHref: string;
  orders: OrderHistoryCardItem[];
}

/**
 * OrderHistoryPagePresenter renders the /orders index route as a card list.
 * Empty state: displays a "no orders yet" message with a CTA back to /products.
 */
export function OrderHistoryPagePresenter({
  title,
  sellerLabel,
  totalLabel,
  emptyMessage,
  browseCtaLabel,
  browseCtaHref,
  orders,
}: OrderHistoryPagePresenterProps) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-2xl font-bold text-gray-900">{title}</h1>

      {orders.length === 0 ? (
        <div className="mt-8 flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
          <svg
            aria-hidden="true"
            className="h-12 w-12 text-gray-300"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 00-3 3h15.75m-12.75-3h11.218a2.25 2.25 0 002.166-1.638l1.824-6.39A1.125 1.125 0 0020.83 4.5H5.106"
            />
          </svg>
          <p className="mt-4 text-sm text-gray-500">{emptyMessage}</p>
          <Link
            href={browseCtaHref}
            className="mt-6 inline-block rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700"
          >
            {browseCtaLabel}
          </Link>
        </div>
      ) : (
        <ul className="mt-6 space-y-4">
          {orders.map((order) => (
            <li key={order.id}>
              <Link
                href={order.href}
                className="block rounded-lg border border-gray-200 bg-white p-4 transition-shadow hover:shadow-md"
              >
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <p className="text-xs text-gray-500">{order.createdAtLabel}</p>
                    <p className="mt-1 text-sm font-medium text-gray-900">
                      {sellerLabel}: {order.sellerName}
                    </p>
                  </div>
                  <span className="inline-flex items-center rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
                    {order.statusLabel}
                  </span>
                </div>
                <div className="mt-3 flex items-center justify-between">
                  <span className="text-xs text-gray-500">{totalLabel}</span>
                  <span className="text-base font-bold text-gray-900">{order.totalLabel}</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
