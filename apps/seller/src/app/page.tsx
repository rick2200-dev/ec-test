import StatsCard from "@/components/StatsCard";
import { salesStats, orders } from "@/lib/mock-data";
import { formatCurrency, STATUS_LABELS, STATUS_COLORS } from "@/lib/utils";

export default function DashboardPage() {
  const recentOrders = orders.slice(0, 5);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">ダッシュボード</h2>
        <p className="text-text-secondary mt-1">ストアの概要を確認できます</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatsCard
          title="今日の売上"
          value={formatCurrency(salesStats.todaySales)}
          subtitle="前日比 +12.5%"
          accent="success"
        />
        <StatsCard
          title="今月の売上"
          value={formatCurrency(salesStats.monthlySales)}
          subtitle="先月比 +8.3%"
          accent="default"
        />
        <StatsCard
          title="未処理注文"
          value={`${salesStats.pendingOrders}件`}
          subtitle="早めに対応してください"
          accent="warning"
        />
        <StatsCard
          title="在庫アラート"
          value={`${salesStats.stockAlerts}件`}
          subtitle="在庫が少ない商品があります"
          accent="danger"
        />
      </div>

      {/* Recent orders */}
      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="px-6 py-4 border-b border-border flex items-center justify-between">
          <h3 className="text-lg font-semibold text-text-primary">最近の注文</h3>
          <a
            href="/orders"
            className="text-sm text-accent hover:text-accent-hover font-medium"
          >
            すべて表示 &rarr;
          </a>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  注文ID
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  購入者
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  金額
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  ステータス
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  日時
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {recentOrders.map((order) => (
                <tr key={order.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-mono text-text-primary">
                    {order.id}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">
                    {order.buyerName}
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {formatCurrency(order.totalAmount)}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[order.status]}`}
                    >
                      {STATUS_LABELS[order.status]}
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
