import type { PointsTransaction, PointsTransactionType } from "@ec-marketplace/types";

export interface PointsTransactionsListPresenterProps {
  transactions: PointsTransaction[];
  emptyLabel: string;
  loadingLabel: string;
  loaded: boolean;
  typeLabels: Record<PointsTransactionType, string>;
  columns: {
    date: string;
    type: string;
    amount: string;
    balanceAfter: string;
    note: string;
  };
  locale: string;
}

export function PointsTransactionsListPresenter({
  transactions,
  emptyLabel,
  loadingLabel,
  loaded,
  typeLabels,
  columns,
  locale,
}: PointsTransactionsListPresenterProps) {
  if (!loaded) {
    return <p className="py-8 text-center text-sm text-gray-500">{loadingLabel}</p>;
  }
  if (transactions.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-gray-300 py-8 text-center text-sm text-gray-500">
        {emptyLabel}
      </p>
    );
  }

  const dtf = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" });

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          <tr>
            <th scope="col" className="px-4 py-3">
              {columns.date}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.type}
            </th>
            <th scope="col" className="px-4 py-3 text-right">
              {columns.amount}
            </th>
            <th scope="col" className="px-4 py-3 text-right">
              {columns.balanceAfter}
            </th>
            <th scope="col" className="px-4 py-3">
              {columns.note}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {transactions.map((tx) => (
            <tr key={tx.id}>
              <td className="px-4 py-3 text-gray-700">{dtf.format(new Date(tx.created_at))}</td>
              <td className="px-4 py-3 text-gray-900">{typeLabels[tx.type] ?? tx.type}</td>
              <td
                className={`px-4 py-3 text-right font-semibold ${
                  tx.amount >= 0 ? "text-green-700" : "text-red-700"
                }`}
              >
                {tx.amount >= 0 ? "+" : ""}
                {tx.amount.toLocaleString()}
              </td>
              <td className="px-4 py-3 text-right text-gray-700">
                {tx.balance_after.toLocaleString()}
              </td>
              <td className="px-4 py-3 text-gray-500">{tx.note ?? ""}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
