"use client";

import { useState } from "react";
import StatusBadge from "@/components/StatusBadge";
import { sellers } from "@/lib/mock-data";

type FilterTab = "all" | "pending" | "approved" | "suspended";

const tabs: { key: FilterTab; label: string }[] = [
  { key: "all", label: "全て" },
  { key: "pending", label: "承認待ち" },
  { key: "approved", label: "承認済み" },
  { key: "suspended", label: "停止中" },
];

export default function SellersPage() {
  const [activeTab, setActiveTab] = useState<FilterTab>("all");

  const filteredSellers =
    activeTab === "all"
      ? sellers
      : sellers.filter((s) => s.status === activeTab);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">セラー管理</h2>
        <p className="text-text-secondary mt-1">プラットフォーム上のセラーを管理します</p>
      </div>

      {/* Filter tabs */}
      <div className="flex gap-1 bg-surface rounded-lg p-1 w-fit">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setActiveTab(tab.key)}
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
                  セラー名
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  テナント
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  ステータス
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  手数料率
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  Stripe接続
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  作成日
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filteredSellers.map((seller) => (
                <tr key={seller.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {seller.name}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {seller.tenantName}
                  </td>
                  <td className="px-6 py-4">
                    <StatusBadge status={seller.status} />
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">
                    {seller.commissionRate}%
                  </td>
                  <td className="px-6 py-4">
                    {seller.stripeConnected ? (
                      <span className="text-success text-sm font-medium">接続済み</span>
                    ) : (
                      <span className="text-text-secondary text-sm">未接続</span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {seller.createdAt}
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex gap-2">
                      {seller.status === "pending" && (
                        <button className="text-sm text-success hover:text-green-700 font-medium">
                          承認
                        </button>
                      )}
                      {seller.status === "approved" && (
                        <button className="text-sm text-danger hover:text-red-700 font-medium">
                          停止
                        </button>
                      )}
                      {seller.status === "suspended" && (
                        <button className="text-sm text-accent hover:text-accent-hover font-medium">
                          再開
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
