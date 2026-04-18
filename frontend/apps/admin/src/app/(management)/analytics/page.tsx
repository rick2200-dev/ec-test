"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";

import { listSellers } from "@/lib/api";

// Aggregate transaction / commission totals have no backing API yet —
// the numbers shown as "—" below go live once an analytics service (or
// an on-demand aggregation endpoint) lands. Active seller count is the
// one metric we can derive honestly from sellers-svc alone.
export default function AnalyticsPage() {
  const t = useTranslations();
  const [activeSellers, setActiveSellers] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const sellers = await listSellers();
      if (cancelled) return;
      setActiveSellers(sellers.filter((s) => s.status === "approved").length);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const metrics: { title: string; value: string; hint?: string }[] = [
    {
      title: t("analytics.monthlyTransaction"),
      value: "—",
      hint: t("analytics.metricUnavailable"),
    },
    {
      title: t("analytics.monthlyCommission"),
      value: "—",
      hint: t("analytics.metricUnavailable"),
    },
    {
      title: t("analytics.averageOrder"),
      value: "—",
      hint: t("analytics.metricUnavailable"),
    },
    {
      title: t("analytics.activeSellers"),
      value: activeSellers == null ? "…" : String(activeSellers),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("analytics.title")}</h2>
        <p className="text-text-secondary mt-1">{t("analytics.description")}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {metrics.map((metric) => (
          <div
            key={metric.title}
            className="bg-white rounded-lg border border-border p-6 shadow-sm"
          >
            <p className="text-sm text-text-secondary">{metric.title}</p>
            <p className="text-2xl font-bold text-text-primary mt-1">{metric.value}</p>
            {metric.hint && <p className="text-xs text-text-secondary mt-2">{metric.hint}</p>}
          </div>
        ))}
      </div>

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
