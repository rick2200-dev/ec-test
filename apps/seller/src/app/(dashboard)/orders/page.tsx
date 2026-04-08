"use client";

import { useState } from "react";
import { orders } from "@/lib/mock-data";
import { formatCurrency, STATUS_COLORS } from "@/lib/utils";
import { useTranslations } from "next-intl";

type StatusFilter = "all" | "pending" | "processing" | "shipped" | "completed";

export default function OrdersPage() {
  const t = useTranslations();
  const [filter, setFilter] = useState<StatusFilter>("all");

  const statusLabels: Record<string, string> = {
    pending: t("order.pending"),
    processing: t("order.processing"),
    shipped: t("order.shipped"),
    completed: t("order.completed"),
    cancelled: t("order.cancelled"),
  };

  const tabs: { key: StatusFilter; labelKey: string }[] = [
    { key: "all", labelKey: "orders.all" },
    { key: "pending", labelKey: "orders.pending" },
    { key: "processing", labelKey: "orders.processing" },
    { key: "shipped", labelKey: "orders.shipped" },
    { key: "completed", labelKey: "orders.completed" },
  ];

  const filteredOrders =
    filter === "all" ? orders : orders.filter((order) => order.status === filter);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("orders.title")}</h2>
        <p className="text-text-secondary mt-1">{t("orders.description")}</p>
      </div>

      {/* Status filter tabs */}
      <div className="flex items-center gap-1 border-b border-border" role="tablist">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setFilter(tab.key)}
            role="tab"
            aria-selected={filter === tab.key}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              filter === tab.key
                ? "border-accent text-accent"
                : "border-transparent text-text-secondary hover:text-text-primary"
            }`}
          >
            {t(tab.labelKey)}
            {tab.key !== "all" && (
              <span className="ml-1.5 text-xs">
                ({orders.filter((o) => o.status === tab.key).length})
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Orders table */}
      <div className="bg-white rounded-lg border border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.orderId")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.date")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.buyer")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.productName")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.amount")}
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  {t("table.status")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {filteredOrders.map((order) => (
                <tr key={order.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-mono text-text-primary">{order.id}</td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {new Date(order.createdAt).toLocaleString("ja-JP", {
                      year: "numeric",
                      month: "2-digit",
                      day: "2-digit",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">{order.buyerName}</td>
                  <td className="px-6 py-4">
                    <div className="text-sm text-text-primary">
                      {order.items.map((item, i) => (
                        <div key={i}>
                          {item.productName}
                          <span className="text-text-secondary"> x{item.quantity}</span>
                        </div>
                      ))}
                    </div>
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {formatCurrency(order.totalAmount)}
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${STATUS_COLORS[order.status]}`}
                    >
                      {statusLabels[order.status]}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {filteredOrders.length === 0 && (
          <div className="px-6 py-12 text-center text-text-secondary" role="status">
            {t("orders.noOrders")}
          </div>
        )}
      </div>
    </div>
  );
}
