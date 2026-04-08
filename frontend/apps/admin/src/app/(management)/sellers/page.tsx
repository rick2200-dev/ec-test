"use client";

import { useState } from "react";
import StatusBadge from "@/components/StatusBadge";
import { sellers } from "@/lib/mock-data";
import { useTranslations } from "next-intl";

type FilterTab = "all" | "pending" | "approved" | "suspended";

export default function SellersPage() {
  const [activeTab, setActiveTab] = useState<FilterTab>("all");
  const t = useTranslations();

  const tabs: { key: FilterTab; label: string }[] = [
    { key: "all", label: t("sellers.all") },
    { key: "pending", label: t("sellers.pending") },
    { key: "approved", label: t("sellers.approved") },
    { key: "suspended", label: t("sellers.suspended") },
  ];

  const filteredSellers =
    activeTab === "all" ? sellers : sellers.filter((s) => s.status === activeTab);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("sellers.title")}</h2>
        <p className="text-text-secondary mt-1">{t("sellers.description")}</p>
      </div>

      {/* Filter tabs */}
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
                  {t("sellers.tenant")}
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
              {filteredSellers.map((seller) => (
                <tr key={seller.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">{seller.name}</td>
                  <td className="px-6 py-4 text-sm text-text-secondary">{seller.tenantName}</td>
                  <td className="px-6 py-4">
                    <StatusBadge status={seller.status} />
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">{seller.commissionRate}%</td>
                  <td className="px-6 py-4">
                    {seller.stripeConnected ? (
                      <span className="text-success text-sm font-medium">
                        {t("sellers.connected")}
                      </span>
                    ) : (
                      <span className="text-text-secondary text-sm">
                        {t("sellers.notConnected")}
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">{seller.createdAt}</td>
                  <td className="px-6 py-4">
                    <div className="flex gap-2">
                      {seller.status === "pending" && (
                        <button
                          className="text-sm text-success hover:text-green-700 font-medium"
                          aria-label={`${t("sellers.approve")} ${seller.name}`}
                        >
                          {t("sellers.approve")}
                        </button>
                      )}
                      {seller.status === "approved" && (
                        <button
                          className="text-sm text-danger hover:text-red-700 font-medium"
                          aria-label={`${t("sellers.suspend")} ${seller.name}`}
                        >
                          {t("sellers.suspend")}
                        </button>
                      )}
                      {seller.status === "suspended" && (
                        <button
                          className="text-sm text-accent hover:text-accent-hover font-medium"
                          aria-label={`${t("sellers.resume")} ${seller.name}`}
                        >
                          {t("sellers.resume")}
                        </button>
                      )}
                    </div>
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
