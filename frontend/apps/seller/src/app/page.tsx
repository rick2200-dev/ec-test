import { salesStats, orders } from "@/lib/mock-data";
import { formatCurrency, STATUS_COLORS } from "@/lib/utils";
import { getTranslations } from "next-intl/server";
import {
  DashboardPagePresenter,
  type DashboardOrderRow,
  type DashboardStatCard,
} from "@/components/pages/DashboardPage/DashboardPage.presenter";

export default async function DashboardPage() {
  const t = await getTranslations();
  const recentOrders = orders.slice(0, 5);

  const statusLabels: Record<string, string> = {
    pending: t("order.pending"),
    processing: t("order.processing"),
    shipped: t("order.shipped"),
    completed: t("order.completed"),
    cancelled: t("order.cancelled"),
  };

  const statsCards: DashboardStatCard[] = [
    {
      id: "today-sales",
      title: t("dashboard.todaySales"),
      value: formatCurrency(salesStats.todaySales),
      subtitle: t("dashboard.subtitleTodaySales"),
      accent: "success",
    },
    {
      id: "monthly-sales",
      title: t("dashboard.monthlySales"),
      value: formatCurrency(salesStats.monthlySales),
      subtitle: t("dashboard.subtitleMonthlySales"),
      accent: "default",
    },
    {
      id: "pending-orders",
      title: t("dashboard.pendingOrders"),
      value: `${salesStats.pendingOrders}件`,
      subtitle: t("dashboard.subtitlePendingOrders"),
      accent: "warning",
    },
    {
      id: "stock-alerts",
      title: t("dashboard.stockAlerts"),
      value: `${salesStats.stockAlerts}件`,
      subtitle: t("dashboard.subtitleStockAlerts"),
      accent: "danger",
    },
  ];

  const orderRows: DashboardOrderRow[] = recentOrders.map((order) => ({
    id: order.id,
    buyerName: order.buyerName,
    amountLabel: formatCurrency(order.totalAmount),
    statusLabel: statusLabels[order.status] ?? order.status,
    statusClassName: STATUS_COLORS[order.status] ?? "",
    dateLabel: new Date(order.createdAt).toLocaleString("ja-JP", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }),
  }));

  return (
    <DashboardPagePresenter
      heading={{
        title: t("dashboard.title"),
        description: t("dashboard.description"),
      }}
      statsCards={statsCards}
      recentOrdersSection={{
        title: t("dashboard.recentOrders"),
        viewAllHref: "/orders",
        viewAllLabel: t("common.viewAll"),
        columnLabels: {
          orderId: t("table.orderId"),
          buyer: t("table.buyer"),
          amount: t("table.amount"),
          status: t("table.status"),
          date: t("table.date"),
        },
        orders: orderRows,
      }}
    />
  );
}
