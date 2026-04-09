import StatusBadge from "@/components/StatusBadge";
import { subscriptionPlans } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";

export default async function PlansPage() {
  const t = await getTranslations();

  const formatPrice = (amount: number) => {
    if (amount === 0) return t("plans.free");
    return `¥${amount.toLocaleString()}`;
  };

  const formatMaxProducts = (count: number) => {
    if (count < 0) return t("plans.unlimited");
    return count.toString();
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{t("plans.title")}</h2>
          <p className="text-text-secondary mt-1">{t("plans.description")}</p>
        </div>
        <a
          href="/plans/new"
          className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
        >
          {t("plans.createNew")}
        </a>
      </div>

      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.planName")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.tier")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.price")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.searchBoost")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.maxProducts")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.promotedResults")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.status")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("plans.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {subscriptionPlans.map((plan) => (
                <tr key={plan.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">{plan.name}</td>
                  <td className="px-6 py-4 text-sm text-text-primary">{plan.tier}</td>
                  <td className="px-6 py-4 text-sm text-text-primary">
                    {formatPrice(plan.price_amount)}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">x{plan.features.search_boost}</td>
                  <td className="px-6 py-4 text-sm text-text-primary">
                    {formatMaxProducts(plan.features.max_products)}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">
                    {plan.features.promoted_results}
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={plan.status} />
                  </td>
                  <td className="px-6 py-4">
                    <button
                      className="text-sm text-accent hover:text-accent-hover font-medium"
                      aria-label={`${t("plans.edit")} ${plan.name}`}
                    >
                      {t("plans.edit")}
                    </button>
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
