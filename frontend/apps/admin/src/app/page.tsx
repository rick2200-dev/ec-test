import { platformStats, pendingApplications, serviceHealth } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";
import {
  AdminDashboardPagePresenter,
  type AdminDashboardStatCard,
  type AdminPendingApplicationRow,
  type AdminServiceHealthRow,
} from "@/components/pages/DashboardPage/DashboardPage.presenter";
import type { StatusBadgePresenterProps } from "@/components/StatusBadge";
import type { StatusType } from "@/components/StatusBadge";

function formatCurrency(amount: number): string {
  return `¥${amount.toLocaleString()}`;
}

function statusToBadge(
  status: StatusType,
  t: (key: string) => string,
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

export default async function AdminDashboardPage() {
  const t = await getTranslations();

  const statsCards: AdminDashboardStatCard[] = [
    {
      id: "tenants",
      title: t("dashboard.totalTenants"),
      value: `${platformStats.totalTenants}`,
      subtitle: "前月比 +2",
    },
    {
      id: "sellers",
      title: t("dashboard.totalSellers"),
      value: `${platformStats.totalSellers}`,
      subtitle: "前月比 +18",
    },
    {
      id: "transactions",
      title: t("dashboard.monthlyTransaction"),
      value: formatCurrency(platformStats.monthlyTransactionAmount),
      subtitle: "前月比 +15.2%",
    },
    {
      id: "commission",
      title: t("dashboard.monthlyCommission"),
      value: formatCurrency(platformStats.monthlyCommissionIncome),
      subtitle: "前月比 +15.2%",
    },
  ];

  const pendingRows: AdminPendingApplicationRow[] = pendingApplications.map((seller) => ({
    id: seller.id,
    name: seller.name,
    tenantName: seller.tenantName,
    createdAtLabel: seller.createdAt,
    badge: statusToBadge(seller.status as StatusType, t),
  }));

  const serviceRows: AdminServiceHealthRow[] = serviceHealth.map((service) => ({
    name: service.name,
    badge: statusToBadge(service.status, t),
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
        columnLabels: {
          sellerName: t("dashboard.sellerName"),
          tenant: t("dashboard.tenant"),
          applicationDate: t("dashboard.applicationDate"),
          status: t("dashboard.status"),
        },
        rows: pendingRows,
      }}
      serviceHealthSection={{
        title: t("dashboard.serviceHealth"),
        services: serviceRows,
      }}
    />
  );
}
