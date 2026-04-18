import Link from "next/link";
import { formatCurrency } from "@ec-marketplace/ui-utils";
import type { CartItem } from "@ec-marketplace/types";

export interface CartPagePresenterProps {
  title: string;
  emptyTitle: string;
  emptyDescription: string;
  browseCta: string;
  itemColumnLabel: string;
  qtyColumnLabel: string;
  unitPriceColumnLabel: string;
  lineTotalColumnLabel: string;
  totalLabel: string;
  checkoutCta: string;
  loadingLabel: string;
  loginRequiredTitle: string;
  loginRequiredCta: string;
  /** Items in the cart. `null` means still loading; `[]` means empty. */
  items: CartItem[] | null;
  /** When set, renders a 401 → login state instead of the cart UI. */
  needsLogin: boolean;
  /** Pre-rendered error banner (reserved for read-side errors). */
  errorSlot?: React.ReactNode;
  /** Pre-rendered actions block on each line (qty/remove buttons in PR 2b). */
  renderLineActions?: (item: CartItem) => React.ReactNode;
  /** Pre-rendered footer actions (clear-cart / checkout buttons). */
  footerActionsSlot?: React.ReactNode;
}

export function CartPagePresenter({
  title,
  emptyTitle,
  emptyDescription,
  browseCta,
  itemColumnLabel,
  qtyColumnLabel,
  unitPriceColumnLabel,
  lineTotalColumnLabel,
  totalLabel,
  checkoutCta,
  loadingLabel,
  loginRequiredTitle,
  loginRequiredCta,
  items,
  needsLogin,
  errorSlot,
  renderLineActions,
  footerActionsSlot,
}: CartPagePresenterProps) {
  if (needsLogin) {
    const returnTo = "/cart";
    return (
      <div className="mx-auto max-w-3xl px-4 py-12 text-center sm:px-6 lg:px-8">
        <h1 className="text-2xl font-bold text-gray-900">{loginRequiredTitle}</h1>
        <a
          href={`/auth/login?returnTo=${encodeURIComponent(returnTo)}`}
          className="mt-6 inline-block rounded-md bg-blue-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-blue-700"
        >
          {loginRequiredCta}
        </a>
      </div>
    );
  }

  if (items === null) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-12 text-center text-sm text-gray-500 sm:px-6 lg:px-8">
        {loadingLabel}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6 lg:px-8">
        <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
        {errorSlot}
        <div className="mt-8 rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
          <p className="text-base text-gray-700">{emptyTitle}</p>
          <p className="mt-1 text-sm text-gray-500">{emptyDescription}</p>
          <Link
            href="/products"
            className="mt-6 inline-block rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {browseCta}
          </Link>
        </div>
      </div>
    );
  }

  const total = items.reduce((sum, item) => sum + item.unit_price_snapshot * item.quantity, 0);

  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
      {errorSlot}

      <div className="mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
            <tr>
              <th scope="col" className="px-4 py-3">
                {itemColumnLabel}
              </th>
              <th scope="col" className="px-4 py-3 text-right">
                {unitPriceColumnLabel}
              </th>
              <th scope="col" className="px-4 py-3 text-right">
                {qtyColumnLabel}
              </th>
              <th scope="col" className="px-4 py-3 text-right">
                {lineTotalColumnLabel}
              </th>
              {renderLineActions && <th scope="col" className="px-4 py-3" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 text-sm">
            {items.map((item) => (
              <tr key={item.sku_id}>
                <td className="px-4 py-4">
                  <div className="font-medium text-gray-900">{item.product_name_snapshot}</div>
                  <div className="text-xs text-gray-500">{item.sku_code_snapshot}</div>
                </td>
                <td className="px-4 py-4 text-right text-gray-700">
                  {formatCurrency(item.unit_price_snapshot)}
                </td>
                <td className="px-4 py-4 text-right text-gray-700">{item.quantity}</td>
                <td className="px-4 py-4 text-right font-medium text-gray-900">
                  {formatCurrency(item.unit_price_snapshot * item.quantity)}
                </td>
                {renderLineActions && (
                  <td className="px-4 py-4 text-right">{renderLineActions(item)}</td>
                )}
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="bg-gray-50 text-sm">
              <td className="px-4 py-3 font-medium text-gray-700" colSpan={3}>
                {totalLabel}
              </td>
              <td className="px-4 py-3 text-right text-base font-bold text-gray-900">
                {formatCurrency(total)}
              </td>
              {renderLineActions && <td className="px-4 py-3" />}
            </tr>
          </tfoot>
        </table>
      </div>

      <div className="mt-6 flex items-center justify-between gap-3">
        <div>{footerActionsSlot}</div>
        <Link
          href="/checkout"
          className="rounded-md bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white hover:bg-blue-700"
        >
          {checkoutCta}
        </Link>
      </div>
    </div>
  );
}
