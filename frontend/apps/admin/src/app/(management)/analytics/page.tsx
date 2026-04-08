import { platformStats } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";

function formatCurrency(amount: number): string {
  return `¥${amount.toLocaleString()}`;
}

export default async function AnalyticsPage() {
  const t = await getTranslations();

  const metrics = [
    {
      title: t("analytics.monthlyTransaction"),
      value: formatCurrency(platformStats.monthlyTransactionAmount),
      change: "+15.2%",
    },
    {
      title: t("analytics.monthlyCommission"),
      value: formatCurrency(platformStats.monthlyCommissionIncome),
      change: "+15.2%",
    },
    { title: t("analytics.averageOrder"), value: formatCurrency(12500), change: "+3.1%" },
    { title: t("analytics.activeSellers"), value: "142", change: "+8" },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("analytics.title")}</h2>
        <p className="text-text-secondary mt-1">{t("analytics.description")}</p>
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
          <h3 className="text-lg font-semibold text-text-primary mb-4">
            {t("analytics.transactionTrend")}
          </h3>
          <div className="h-64 bg-surface rounded-lg flex items-center justify-center">
            <p className="text-text-secondary text-sm">{t("analytics.chartPlaceholder")}</p>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-border shadow-sm p-6">
          <h3 className="text-lg font-semibold text-text-primary mb-4">
            {t("analytics.salesBySeller")}
          </h3>
          <div className="h-64 bg-surface rounded-lg flex items-center justify-center">
            <p className="text-text-secondary text-sm">{t("analytics.chartPlaceholder")}</p>
          </div>
        </div>

        <div className="bg-white rounded-lg border border-border shadow-sm p-6 lg:col-span-2">
          <h3 className="text-lg font-semibold text-text-primary mb-4">
            {t("analytics.salesByCategory")}
          </h3>
          <div className="h-64 bg-surface rounded-lg flex items-center justify-center">
            <p className="text-text-secondary text-sm">{t("analytics.chartPlaceholder")}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
