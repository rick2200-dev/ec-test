import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { listAdminCoupons } from "@/lib/api";

export const dynamic = "force-dynamic";

function formatDiscount(c: {
  discount_type: "percent" | "fixed_amount";
  discount_percent_bps?: number | null;
  discount_amount?: number | null;
  currency: string;
}): string {
  if (c.discount_type === "percent" && c.discount_percent_bps != null) {
    return `${(c.discount_percent_bps / 100).toFixed(2)}%`;
  }
  if (c.discount_type === "fixed_amount" && c.discount_amount != null) {
    return `¥${c.discount_amount.toLocaleString()}`;
  }
  return "—";
}

export default async function CouponsPage() {
  const t = await getTranslations("coupons");
  const { coupons } = await listAdminCoupons({ limit: 50 });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{t("title")}</h2>
          <p className="mt-1 text-text-secondary">{t("description")}</p>
        </div>
        <Link
          href="/coupons/new"
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-hover"
        >
          {t("createNew")}
        </Link>
      </div>

      <div className="overflow-hidden rounded-lg border border-border bg-white shadow-sm">
        <table className="min-w-full divide-y divide-border text-sm">
          <thead className="bg-surface text-left text-xs font-medium uppercase tracking-wider text-text-secondary">
            <tr>
              <th className="px-4 py-3">{t("columns.code")}</th>
              <th className="px-4 py-3">{t("columns.title")}</th>
              <th className="px-4 py-3">{t("columns.discount")}</th>
              <th className="px-4 py-3">{t("columns.minOrder")}</th>
              <th className="px-4 py-3">{t("columns.usage")}</th>
              <th className="px-4 py-3">{t("columns.status")}</th>
              <th className="px-4 py-3 text-right">{t("columns.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {coupons.length === 0 && (
              <tr>
                <td colSpan={7} className="px-4 py-10 text-center text-text-secondary">
                  {t("empty")}
                </td>
              </tr>
            )}
            {coupons.map((c) => {
              const isRevoked = c.status === "revoked";
              const usage = c.usage_limit_total
                ? `${c.usage_count} / ${c.usage_limit_total}`
                : String(c.usage_count);
              return (
                <tr
                  key={c.id}
                  className={`transition-colors hover:bg-surface-hover ${
                    isRevoked ? "opacity-60" : ""
                  }`}
                >
                  <td className="px-4 py-3 font-mono text-text-primary">{c.code}</td>
                  <td className="px-4 py-3 text-text-primary">{c.title}</td>
                  <td className="px-4 py-3 text-text-primary">{formatDiscount(c)}</td>
                  <td className="px-4 py-3 text-text-secondary">
                    ¥{c.min_order_amount.toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-text-secondary">{usage}</td>
                  <td className="px-4 py-3">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                        isRevoked ? "bg-gray-200 text-gray-700" : "bg-green-100 text-green-800"
                      }`}
                    >
                      {isRevoked ? t("status.revoked") : t("status.active")}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Link
                      href={`/coupons/${c.id}`}
                      className="text-sm font-medium text-accent hover:text-accent-hover"
                    >
                      {t("actions.detail")}
                    </Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
