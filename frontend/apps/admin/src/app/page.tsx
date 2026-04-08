import StatusBadge from "@/components/StatusBadge";
import { platformStats, pendingApplications, serviceHealth } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";

function formatCurrency(amount: number): string {
  return `¥${amount.toLocaleString()}`;
}

export default async function AdminDashboardPage() {
  const t = await getTranslations();

  const statsCards = [
    {
      title: t("dashboard.totalTenants"),
      value: `${platformStats.totalTenants}`,
      subtitle: "前月比 +2",
    },
    {
      title: t("dashboard.totalSellers"),
      value: `${platformStats.totalSellers}`,
      subtitle: "前月比 +18",
    },
    {
      title: t("dashboard.monthlyTransaction"),
      value: formatCurrency(platformStats.monthlyTransactionAmount),
      subtitle: "前月比 +15.2%",
    },
    {
      title: t("dashboard.monthlyCommission"),
      value: formatCurrency(platformStats.monthlyCommissionIncome),
      subtitle: "前月比 +15.2%",
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("dashboard.title")}</h2>
        <p className="text-text-secondary mt-1">{t("dashboard.description")}</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statsCards.map((card) => (
          <div key={card.title} className="bg-white rounded-lg border border-border p-6 shadow-sm">
            <p className="text-sm text-text-secondary">{card.title}</p>
            <p className="text-2xl font-bold text-text-primary mt-1">{card.value}</p>
            <p className="text-xs text-text-secondary mt-2">{card.subtitle}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Pending seller applications */}
        <div className="bg-white rounded-lg border border-border shadow-sm">
          <div className="px-6 py-4 border-b border-border flex items-center justify-between">
            <h3 className="text-lg font-semibold text-text-primary">
              {t("dashboard.pendingApplications")}
            </h3>
            <a href="/sellers" className="text-sm text-accent hover:text-accent-hover font-medium">
              {t("common.viewAll")} &rarr;
            </a>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-surface">
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {t("dashboard.sellerName")}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {t("dashboard.tenant")}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {t("dashboard.applicationDate")}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {t("dashboard.status")}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {pendingApplications.map((seller) => (
                  <tr key={seller.id} className="hover:bg-surface-hover transition-colors">
                    <td className="px-6 py-4 text-sm font-medium text-text-primary">
                      {seller.name}
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{seller.tenantName}</td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{seller.createdAt}</td>
                    <td className="px-6 py-4">
                      <StatusBadge status={seller.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Platform health */}
        <div className="bg-white rounded-lg border border-border shadow-sm">
          <div className="px-6 py-4 border-b border-border">
            <h3 className="text-lg font-semibold text-text-primary">
              {t("dashboard.serviceHealth")}
            </h3>
          </div>
          <div className="p-6 space-y-4">
            {serviceHealth.map((service) => (
              <div key={service.name} className="flex items-center justify-between">
                <span className="text-sm text-text-primary">{service.name}</span>
                <StatusBadge status={service.status} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
