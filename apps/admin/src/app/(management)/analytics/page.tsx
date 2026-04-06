import { platformStats } from "@/lib/mock-data";

function formatCurrency(amount: number): string {
  return `¥${amount.toLocaleString()}`;
}

export default function AnalyticsPage() {
  const metrics = [
    { title: "月間取引額", value: formatCurrency(platformStats.monthlyTransactionAmount), change: "+15.2%" },
    { title: "月間手数料収入", value: formatCurrency(platformStats.monthlyCommissionIncome), change: "+15.2%" },
    { title: "平均注文金額", value: formatCurrency(12500), change: "+3.1%" },
    { title: "アクティブセラー数", value: "142", change: "+8" },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">プラットフォーム分析</h2>
        <p className="text-text-secondary mt-1">プラットフォーム全体のパフォーマンスを分析します</p>
      </div>

      {/* Key metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {metrics.map((metric) => (
          <div
            key={metric.title}
            className="bg-white rounded-lg border border-border p-6 shadow-sm"
          >
            <p className="text-sm text-text-secondary">{metric.title}</p>
            <p className="text-2xl font-bold text-text-primary mt-1">{metric.value}</p>
            <p className="text-xs text-success mt-2">{metric.change}</p>
          </div>
        ))}
      </div>

      {/* Chart placeholders */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-lg border border-border shadow-sm p-6">
          <h3 className="text-lg font-semibold text-text-primary mb-4">取引額推移</h3>
          <div className="h-64 bg-surface rounded-lg flex items-center justify-center">
            <p className="text-text-secondary text-sm">チャートコンポーネント（実装予定）</p>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-border shadow-sm p-6">
          <h3 className="text-lg font-semibold text-text-primary mb-4">セラー別売上</h3>
          <div className="h-64 bg-surface rounded-lg flex items-center justify-center">
            <p className="text-text-secondary text-sm">チャートコンポーネント（実装予定）</p>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-border shadow-sm p-6 lg:col-span-2">
          <h3 className="text-lg font-semibold text-text-primary mb-4">カテゴリ別売上</h3>
          <div className="h-64 bg-surface rounded-lg flex items-center justify-center">
            <p className="text-text-secondary text-sm">チャートコンポーネント（実装予定）</p>
          </div>
        </div>
      </div>
    </div>
  );
}
