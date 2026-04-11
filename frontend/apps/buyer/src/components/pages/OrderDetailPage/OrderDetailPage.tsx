import { notFound } from "next/navigation";
import { getTranslations, getLocale } from "next-intl/server";
import { getBuyerOrderById, canContactSeller } from "@/lib/mock-orders";
import { formatPrice } from "@/lib/mock-data";
import StartInquiryButton from "@/components/StartInquiryButton";
import {
  OrderDetailPagePresenter,
  type OrderDetailLineRow,
} from "./OrderDetailPage.presenter";

export interface OrderDetailPageProps {
  orderId: string;
}

export default async function OrderDetailPage({ orderId }: OrderDetailPageProps) {
  const order = getBuyerOrderById(orderId);
  if (!order) {
    notFound();
  }

  const t = await getTranslations("orders");
  const locale = await getLocale();
  const contactable = canContactSeller(order.status);

  const lines: OrderDetailLineRow[] = order.lines.map((line) => ({
    key: line.sku_id,
    productName: line.product_name,
    skuCode: line.sku_code,
    sellerName: line.seller_name,
    quantity: line.quantity,
    unitPriceLabel: formatPrice(line.unit_price, order.currency),
    subtotalLabel: formatPrice(line.unit_price * line.quantity, order.currency),
    actionSlot: contactable ? (
      <StartInquiryButton
        sellerId={line.seller_id}
        skuId={line.sku_id}
        productName={line.product_name}
        skuCode={line.sku_code}
      />
    ) : undefined,
  }));

  return (
    <OrderDetailPagePresenter
      backHref="/orders"
      backLabel={t("title")}
      orderId={order.id}
      orderIdLabel={t("orderId")}
      orderedAt={new Date(order.ordered_at).toLocaleString(locale, {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      })}
      orderedAtLabel={t("orderedAt")}
      statusLabel={t(`status_${order.status}`)}
      totalLabel={t("total")}
      totalValue={formatPrice(order.total_amount, order.currency)}
      productsLabel={t("products")}
      purchaseRequiredNotice={contactable ? undefined : t("purchaseRequiredNotice")}
      lines={lines}
    />
  );
}
