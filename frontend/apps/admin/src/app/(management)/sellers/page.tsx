"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";

import StatusBadge from "@/components/StatusBadge";
import { approveAdminSeller, listSellers } from "@/lib/api";
import type { Seller } from "@/lib/types";

type FilterTab = "all" | "pending" | "approved" | "suspended";

function formatRateBps(bps: number): string {
  if (!Number.isFinite(bps) || bps <= 0) return "-";
  return `${(bps / 100).toFixed(2).replace(/\.?0+$/, "")}%`;
}

function formatDate(iso: string): string {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString();
}

export default function SellersPage() {
  const t = useTranslations();
  const [activeTab, setActiveTab] = useState<FilterTab>("all");
  const [sellers, setSellers] = useState<Seller[]>([]);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    const items = await listSellers();
    setSellers(items);
    setLoading(false);
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  const handleApprove = async (seller: Seller) => {
    if (busyId) return;
    setBusyId(seller.id);
    try {
      await approveAdminSeller(seller.id);
      await reload();
    } finally {
      setBusyId(null);
    }
  };

  const tabs: { key: FilterTab; label: string }[] = [
    { key: "all", label: t("sellers.all") },
    { key: "pending", label: t("sellers.pending") },
    { key: "approved", label: t("sellers.approved") },
    { key: "suspended", label: t("sellers.suspended") },
  ];

  const filteredSellers = sellers.filter((s) => {
    if (activeTab === "all") return true;
    if (activeTab === "suspended") return s.status === "suspended" || s.status === "rejected";
    return s.status === activeTab;
  });

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("sellers.title")}</h2>
        <p className="text-text-secondary mt-1">{t("sellers.description")}</p>
      </div>

      <div className="flex gap-1 bg-surface rounded-lg p-1 w-fit" role="tablist">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
            role="tab"
            aria-selected={activeTab === tab.key}
            className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
              activeTab === tab.key
                ? "bg-white text-text-primary shadow-sm"
                : "text-text-secondary hover:text-text-primary"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("sellers.sellerName")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("sellers.status")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("sellers.commissionRate")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("sellers.stripeConnection")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("sellers.createdDate")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("sellers.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {loading && (
                <tr>
                  <td colSpan={6} className="px-6 py-6 text-center text-sm text-text-secondary">
                    {t("sellers.loading")}
                  </td>
                </tr>
              )}
              {!loading && filteredSellers.length === 0 && (
                <tr>
                  <td colSpan={6} className="px-6 py-6 text-center text-sm text-text-secondary">
                    {t("sellers.empty")}
                  </td>
                </tr>
              )}
              {!loading &&
                filteredSellers.map((seller) => {
                  const stripeConnected = seller.stripe_account_id !== "";
                  const displayStatus =
                    seller.status === "rejected" ? "suspended" : seller.status;
                  return (
                    <tr key={seller.id} className="hover:bg-surface-hover transition-colors">
                      <td className="px-6 py-4 text-sm font-medium text-text-primary">
                        {seller.name}
                      </td>
                      <td className="px-6 py-4">
                        <StatusBadge status={displayStatus} />
                      </td>
                      <td className="px-6 py-4 text-sm text-text-primary">
                        {formatRateBps(seller.commission_rate_bps)}
                      </td>
                      <td className="px-6 py-4">
                        {stripeConnected ? (
                          <span className="text-success text-sm font-medium">
                            {t("sellers.connected")}
                          </span>
                        ) : (
                          <span className="text-text-secondary text-sm">
                            {t("sellers.notConnected")}
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-sm text-text-secondary">
                        {formatDate(seller.created_at)}
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex gap-2">
                          {seller.status === "pending" && (
                            <button
                              onClick={() => handleApprove(seller)}
                              disabled={busyId === seller.id}
                              className="text-sm text-success hover:text-green-700 font-medium disabled:opacity-50"
                              aria-label={`${t("sellers.approve")} ${seller.name}`}
                            >
                              {busyId === seller.id ? t("sellers.approving") : t("sellers.approve")}
                            </button>
                          )}
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
