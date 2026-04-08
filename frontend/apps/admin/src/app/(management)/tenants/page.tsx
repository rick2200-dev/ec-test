import StatusBadge from "@/components/StatusBadge";
import { tenants } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";

export default async function TenantsPage() {
  const t = await getTranslations();

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{t("tenants.title")}</h2>
          <p className="text-text-secondary mt-1">{t("tenants.description")}</p>
        </div>
        <a
          href="/tenants/new"
          className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
        >
          {t("tenants.createNew")}
        </a>
      </div>

      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("tenants.tenantName")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("tenants.slug")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("tenants.status")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("tenants.sellerCount")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("tenants.createdDate")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("tenants.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {tenants.map((tenant) => (
                <tr key={tenant.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">{tenant.name}</td>
                  <td className="px-6 py-4 text-sm font-mono text-text-secondary">{tenant.slug}</td>
                  <td className="px-6 py-4">
                    <StatusBadge status={tenant.status} />
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">{tenant.sellerCount}</td>
                  <td className="px-6 py-4 text-sm text-text-secondary">{tenant.createdAt}</td>
                  <td className="px-6 py-4">
                    <button
                      className="text-sm text-accent hover:text-accent-hover font-medium"
                      aria-label={`${t("tenants.edit")} ${tenant.name}`}
                    >
                      {t("tenants.edit")}
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
