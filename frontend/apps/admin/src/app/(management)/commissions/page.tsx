import { commissionRules } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";

export default async function CommissionsPage() {
  const t = await getTranslations();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{t("commissions.title")}</h2>
          <p className="text-text-secondary mt-1">{t("commissions.description")}</p>
        </div>
        <a
          href="/commissions/new"
          className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
        >
          {t("commissions.createNew")}
        </a>
      </div>

      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("commissions.tenant")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("commissions.seller")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("commissions.category")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("commissions.rate")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("commissions.priority")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("commissions.validPeriod")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {commissionRules.map((rule) => (
                <tr key={rule.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {rule.tenantName}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {rule.sellerName ?? t("commissions.allSellers")}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {rule.category ?? t("commissions.allCategories")}
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">{rule.rate}%</td>
                  <td className="px-6 py-4 text-sm text-text-primary">{rule.priority}</td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {rule.validFrom} ~ {rule.validUntil ?? t("commissions.unlimited")}
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
