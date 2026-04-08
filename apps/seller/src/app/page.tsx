import StatsCard from "@/components/StatsCard";
import { salesStats, orders } from "@/lib/mock-data";
import { formatCurrency, STATUS_COLORS } from "@/lib/utils";
import { getTranslations } from "next-intl/server";

export default async function DashboardPage() {
  const t = await getTranslations();
  const recentOrders = orders.slice(0, 5);

  const statusLabels: Record<string, string> = {
    pending: t("order.pending"),
    processing: t("order.processing"),
    shipped: t("order.shipped"),
    completed: t("order.completed"),
    cancelled: t("order.cancelled"),
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("dashboard.title")}</h2>
        <p className="text-text-secondary mt-1">{t("dashboard.description")}</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatsCard
          title={t("dashboard.todaySales")}
          value={formatCurrency(salesStats.todaySales)}
          subtitle={t("dashboard.subtitleTodaySales")}
          accent="success"
        />
        <StatsCard
          title={t("dashboard.monthlySales")}
          value={formatCurrency(salesStats.monthlySales)}
          subtitle={t("dashboard.subtitleMonthlySales")}
          accent="default"
        />
        <StatsCard
          title={t("dashboard.pendingOrders")}
          value={`${salesStats.pendingOrders}件`}
          subtitle={t("dashboard.subtitlePendingOrders")}
          accent="warning"
        />
        <StatsCard
          title={t("dashboard.stockAlerts")}
          value={`${salesStats.stockAlerts}件`}
          subtitle={t("dashboard.subtitleStockAlerts")}
          accent="danger"
        />
      </div>

      {/* Recent orders */}
      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="px-6 py-4 border-b border-border flex items-center justify-between">
          <h3 className="text-lg font-semibold text-text-primary">{t("dashboard.recentOrders")}</h3>
          <a href="/orders" className="text-sm text-accent hover:text-accent-hover font-medium">
            {t("common.viewAll")} &rarr;
          </a>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.orderId")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.buyer")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.amount")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.status")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.date")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {recentOrders.map((order) => (
                <tr key={order.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-mono text-text-primary">{order.id}</td>
                  <td className="px-6 py-4 text-sm text-text-primary">{order.buyerName}</td>
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {formatCurrency(order.totalAmount)}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[order.status]}`}
                    >
                      {statusLabels[order.status]}
                    </span>
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {new Date(order.createdAt).toLocaleString("ja-JP", {
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
