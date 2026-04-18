"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";

import { listAdminCommissions, listSellers } from "@/lib/api";
import type { CommissionRule, Seller } from "@/lib/types";

function formatRate(bps: number): string {
  if (!Number.isFinite(bps) || bps < 0) return "-";
  return `${(bps / 100).toFixed(2).replace(/\.?0+$/, "")}%`;
}

function formatDate(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString();
}

export default function CommissionsPage() {
  const t = useTranslations();
  const [rules, setRules] = useState<CommissionRule[]>([]);
  const [sellerById, setSellerById] = useState<Record<string, Seller>>({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      const [rs, sellers] = await Promise.all([listAdminCommissions(), listSellers()]);
      if (cancelled) return;
      const byId: Record<string, Seller> = {};
      for (const s of sellers) byId[s.id] = s;
      setRules(rs);
      setSellerById(byId);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

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
              {loading && (
                <tr>
                  <td colSpan={5} className="px-6 py-6 text-center text-sm text-text-secondary">
                    {t("commissions.loading")}
                  </td>
                </tr>
              )}
              {!loading && rules.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-6 py-6 text-center text-sm text-text-secondary">
                    {t("commissions.empty")}
                  </td>
                </tr>
              )}
              {!loading &&
                rules.map((rule) => {
                  const sellerName =
                    rule.seller_id && sellerById[rule.seller_id]
                      ? sellerById[rule.seller_id]!.name
                      : t("commissions.allSellers");
                  return (
                    <tr key={rule.id} className="hover:bg-surface-hover transition-colors">
                      <td className="px-6 py-4 text-sm text-text-primary">{sellerName}</td>
                      <td className="px-6 py-4 text-sm text-text-secondary">
                        {rule.category_id ?? t("commissions.allCategories")}
                      </td>
                      <td className="px-6 py-4 text-sm font-medium text-text-primary">
                        {formatRate(rule.rate_bps)}
                      </td>
                      <td className="px-6 py-4 text-sm text-text-primary">{rule.priority}</td>
                      <td className="px-6 py-4 text-sm text-text-secondary">
                        {formatDate(rule.valid_from)} ~{" "}
                        {rule.valid_until ? formatDate(rule.valid_until) : t("commissions.unlimited")}
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
