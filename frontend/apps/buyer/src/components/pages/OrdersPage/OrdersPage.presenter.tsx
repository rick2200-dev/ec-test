import Link from "next/link";

export interface OrdersPageOrderItem {
  id: string;
  href: string;
  orderedAt: string;
  totalLabel: string;
  statusLabel: string;
  statusTone: "pending" | "paid" | "shipped" | "completed" | "cancelled";
  productSummary: string;
}

export interface OrdersPagePresenterProps {
  title: string;
  description: string;
  emptyLabel: string;
  orderIdLabel: string;
  orderedAtLabel: string;
  totalLabel: string;
  statusLabel: string;
  productsLabel: string;
  actionsLabel: string;
  viewDetailLabel: string;
  orders: OrdersPageOrderItem[];
}

const TONE_CLASS: Record<OrdersPageOrderItem["statusTone"], string> = {
  pending: "bg-yellow-100 text-yellow-800",
  paid: "bg-blue-100 text-blue-800",
  shipped: "bg-indigo-100 text-indigo-800",
  completed: "bg-green-100 text-green-800",
  cancelled: "bg-gray-100 text-gray-600",
};

export function OrdersPagePresenter({
  title,
  description,
  emptyLabel,
  orderIdLabel,
  orderedAtLabel,
  totalLabel,
  statusLabel,
  productsLabel,
  actionsLabel,
  viewDetailLabel,
  orders,
}: OrdersPagePresenterProps) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <header>
        <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
        <p className="mt-1 text-sm text-gray-500">{description}</p>
      </header>

      {orders.length === 0 ? (
        <div
          className="mt-6 rounded-lg border border-dashed border-gray-300 bg-white py-12 text-center text-sm text-gray-500"
          role="status"
        >
          {emptyLabel}
        </div>
      ) : (
        <div className="mt-6 overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50 text-left">
                <th className="px-4 py-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  {orderIdLabel}
                </th>
                <th className="px-4 py-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  {orderedAtLabel}
                </th>
                <th className="px-4 py-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  {productsLabel}
                </th>
                <th className="px-4 py-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  {totalLabel}
                </th>
                <th className="px-4 py-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  {statusLabel}
                </th>
                <th className="px-4 py-3 text-xs font-medium uppercase tracking-wider text-gray-500">
                  {actionsLabel}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {orders.map((order) => (
                <tr key={order.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-mono text-xs text-gray-600">{order.id}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{order.orderedAt}</td>
                  <td className="px-4 py-3 text-sm text-gray-900">{order.productSummary}</td>
                  <td className="px-4 py-3 text-sm font-medium text-gray-900">
                    {order.totalLabel}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                        TONE_CLASS[order.statusTone]
                      }`}
                    >
                      {order.statusLabel}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <Link
                      href={order.href}
                      className="text-sm font-medium text-blue-600 hover:text-blue-800"
                    >
                      {viewDetailLabel}
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
