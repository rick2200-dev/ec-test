import {
  StatsCardPresenter,
  type StatsCardPresenterProps,
} from "../../StatsCard/StatsCard.presenter";

export interface DashboardStatCard extends StatsCardPresenterProps {
  id: string;
}

export interface DashboardOrderRow {
  id: string;
  buyerName: string;
  amountLabel: string;
  statusLabel: string;
  statusClassName: string;
  dateLabel: string;
}

export interface DashboardPagePresenterProps {
  heading: { title: string; description: string };
  statsCards: DashboardStatCard[];
  recentOrdersSection: {
    title: string;
    viewAllHref: string;
    viewAllLabel: string;
    columnLabels: {
      orderId: string;
      buyer: string;
      amount: string;
      status: string;
      date: string;
    };
    orders: DashboardOrderRow[];
  };
}

export function DashboardPagePresenter({
  heading,
  statsCards,
  recentOrdersSection,
}: DashboardPagePresenterProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{heading.title}</h2>
        <p className="text-text-secondary mt-1">{heading.description}</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statsCards.map(({ id, ...cardProps }) => (
          <StatsCardPresenter key={id} {...cardProps} />
        ))}
      </div>

      {/* Recent orders */}
      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="px-6 py-4 border-b border-border flex items-center justify-between">
          <h3 className="text-lg font-semibold text-text-primary">{recentOrdersSection.title}</h3>
          <a
            href={recentOrdersSection.viewAllHref}
            className="text-sm text-accent hover:text-accent-hover font-medium"
          >
            {recentOrdersSection.viewAllLabel} &rarr;
          </a>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {recentOrdersSection.columnLabels.orderId}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {recentOrdersSection.columnLabels.buyer}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {recentOrdersSection.columnLabels.amount}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {recentOrdersSection.columnLabels.status}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {recentOrdersSection.columnLabels.date}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {recentOrdersSection.orders.map((order) => (
                <tr key={order.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-mono text-text-primary">{order.id}</td>
                  <td className="px-6 py-4 text-sm text-text-primary">{order.buyerName}</td>
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {order.amountLabel}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${order.statusClassName}`}
                    >
                      {order.statusLabel}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">{order.dateLabel}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
