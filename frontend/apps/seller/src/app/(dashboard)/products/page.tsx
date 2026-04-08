import Link from "next/link";
import { products } from "@/lib/mock-data";
import { formatCurrency, STATUS_COLORS } from "@/lib/utils";
import { getTranslations } from "next-intl/server";

export default async function ProductsPage() {
  const t = await getTranslations();

  const statusLabels: Record<string, string> = {
    draft: t("products.draft"),
    active: t("products.published"),
    archived: t("products.archived"),
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{t("products.title")}</h2>
          <p className="text-text-secondary mt-1">{t("products.description")}</p>
        </div>
        <Link
          href="/products/new"
          className="inline-flex items-center gap-2 bg-accent hover:bg-accent-hover text-white px-4 py-2.5 rounded-lg text-sm font-medium transition-colors"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          {t("products.newProduct")}
        </Link>
      </div>

      {/* Search and filter */}
      <div className="flex items-center gap-4">
        <div className="flex-1 relative">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-secondary"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
          <input
            type="text"
            placeholder={t("products.searchPlaceholder")}
            aria-label={t("products.searchPlaceholder")}
            className="w-full pl-10 pr-4 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
          />
        </div>
        <select className="border border-border rounded-lg px-3 py-2.5 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent">
          <option value="">{t("products.allStatuses")}</option>
          <option value="active">{t("products.published")}</option>
          <option value="draft">{t("products.draft")}</option>
          <option value="archived">{t("products.archived")}</option>
        </select>
      </div>

      {/* Products table */}
      <div className="bg-white rounded-lg border border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.productName")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.skuCount")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.status")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.price")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.inventory")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {products.map((product) => {
                const totalStock = product.skus.reduce((sum, sku) => sum + sku.stockQuantity, 0);
                const priceRange =
                  product.skus.length > 0
                    ? formatCurrency(Math.min(...product.skus.map((s) => s.price)))
                    : "-";

                return (
                  <tr key={product.id} className="hover:bg-surface-hover transition-colors">
                    <td className="px-6 py-4">
                      <div>
                        <p className="text-sm font-medium text-text-primary">{product.name}</p>
                        <p className="text-xs text-text-secondary mt-0.5">{product.slug}</p>
                      </div>
                    </td>
                    <td className="px-6 py-4 text-sm text-text-primary">{product.skus.length}</td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[product.status]}`}
                      >
                        {statusLabels[product.status]}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm font-medium text-text-primary">
                      {priceRange}
                    </td>
                    <td className="px-6 py-4 text-sm text-text-primary">
                      <span className={totalStock <= 10 ? "text-danger font-medium" : ""}>
                        {totalStock}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <button className="text-sm text-accent hover:text-accent-hover font-medium">
                          {t("products.edit")}
                        </button>
                        <button className="text-sm text-text-secondary hover:text-danger font-medium">
                          {t("products.delete")}
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
