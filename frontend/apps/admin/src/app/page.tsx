"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";

import {
  AdminDashboardPagePresenter,
  type AdminDashboardStatCard,
  type AdminPendingApplicationRow,
} from "@/components/pages/DashboardPage/DashboardPage.presenter";
import type { StatusBadgePresenterProps } from "@/components/StatusBadge";
import type { StatusType } from "@/components/StatusBadge";
import { listSellers } from "@/lib/api";
import type { Seller } from "@/lib/types";

function statusToBadge(
  status: StatusType,
  t: (key: string) => string
): StatusBadgePresenterProps {
  const toneMap: Record<StatusType, StatusBadgePresenterProps["tone"]> = {
    active: "success",
    archived: "neutral",
    approved: "success",
    operational: "success",
    pending: "warning",
    degraded: "warning",
    suspended: "danger",
    down: "danger",
  };
  const labelKeyMap: Record<StatusType, string> = {
    active: "status.active",
    archived: "status.archived",
    approved: "status.approved",
    pending: "status.pending",
    suspended: "status.suspended",
    operational: "status.operational",
    degraded: "status.degraded",
    down: "status.down",
  };
  return {
    tone: toneMap[status] ?? "neutral",
    label: t(labelKeyMap[status]),
  };
}

function formatDate(iso: string): string {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString();
}

// Dashboard pulls what it can from sellers-svc (total + pending list).
// Monthly transaction / commission totals have no aggregation API yet
// — surfaced as em-dashes with a "集計未実装" subtitle so the page is
// honest about the state rather than fabricating numbers.
export default function AdminDashboardPage() {
  const t = useTranslations();
  const [sellers, setSellers] = useState<Seller[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const list = await listSellers();
      if (!cancelled) setSellers(list);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const totalSellers = sellers == null ? "…" : String(sellers.length);
  const pending = sellers == null ? [] : sellers.filter((s) => s.status === "pending");

  const statsCards: AdminDashboardStatCard[] = [
    {
      id: "sellers",
      title: t("dashboard.totalSellers"),
      value: totalSellers,
      subtitle: "",
    },
    {
      id: "transactions",
      title: t("dashboard.monthlyTransaction"),
      value: "—",
      subtitle: t("dashboard.metricUnavailable"),
    },
    {
      id: "commission",
      title: t("dashboard.monthlyCommission"),
      value: "—",
      subtitle: t("dashboard.metricUnavailable"),
    },
  ];

  const pendingRows: AdminPendingApplicationRow[] = pending.map((seller) => ({
    id: seller.id,
    name: seller.name,
    createdAtLabel: formatDate(seller.created_at),
    badge: statusToBadge("pending", t),
  }));

  return (
    <AdminDashboardPagePresenter
      heading={{
        title: t("dashboard.title"),
        description: t("dashboard.description"),
      }}
      statsCards={statsCards}
      pendingSection={{
        title: t("dashboard.pendingApplications"),
        viewAllHref: "/sellers",
        viewAllLabel: t("common.viewAll"),
        emptyLabel: t("dashboard.noPending"),
        columnLabels: {
          sellerName: t("dashboard.sellerName"),
          applicationDate: t("dashboard.applicationDate"),
          status: t("dashboard.status"),
        },
        rows: pendingRows,
      }}
      // serviceHealthSection intentionally omitted: there's no real
      // health registry yet, and the previously-hardcoded list was
      // misleading. Reintroduce once the gateway health check actually
      // probes downstreams.
    />
  );
}
