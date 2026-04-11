import { getTranslations, getLocale } from "next-intl/server";
import { buyerOrders, type BuyerOrderStatus } from "@/lib/mock-orders";
import { formatPrice } from "@/lib/mock-data";
import {
  OrdersPagePresenter,
  type OrdersPageOrderItem,
} from "./OrdersPage.presenter";

const TONE_BY_STATUS: Record<BuyerOrderStatus, OrdersPageOrderItem["statusTone"]> = {
  pending: "pending",
  paid: "paid",
  processing: "paid",
  shipped: "shipped",
  delivered: "shipped",
  completed: "completed",
  cancelled: "cancelled",
};

export default async function OrdersPage() {
  const t = await getTranslations("orders");
  const locale = await getLocale();

  const orderItems: OrdersPageOrderItem[] = buyerOrders.map((order) => {
    const productSummary =
      order.lines.length === 1
        ? `${order.lines[0].product_name} x${order.lines[0].quantity}`
        : `${order.lines[0].product_name} ほか ${order.lines.length - 1} 点`;
    return {
      id: order.id,
      href: `/orders/${order.id}`,
      orderedAt: new Date(order.ordered_at).toLocaleString(locale, {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
      }),
      totalLabel: formatPrice(order.total_amount, order.currency),
      statusLabel: t(`status_${order.status}`),
      statusTone: TONE_BY_STATUS[order.status],
      productSummary,
    };
  });

  return (
    <OrdersPagePresenter
      title={t("title")}
      description={t("description")}
      emptyLabel={t("empty")}
      orderIdLabel={t("orderId")}
      orderedAtLabel={t("orderedAt")}
      totalLabel={t("total")}
      statusLabel={t("status")}
      productsLabel={t("products")}
      actionsLabel="操作"
      viewDetailLabel={t("viewDetail")}
      orders={orderItems}
    />
  );
}
