import type { Inquiry } from "@/lib/types";
import { InquiryListPresenter, type InquiryListItem } from "./InquiryList.presenter";

export interface InquiryListProps {
  inquiries: Inquiry[];
  title: string;
  description: string;
  emptyLabel: string;
  productColumnLabel: string;
  lastMessageColumnLabel: string;
  statusColumnLabel: string;
  unreadLabel: string;
  openLabel: string;
  closedLabel: string;
  locale: string;
}

export default function InquiryList({
  inquiries,
  title,
  description,
  emptyLabel,
  productColumnLabel,
  lastMessageColumnLabel,
  statusColumnLabel,
  unreadLabel,
  openLabel,
  closedLabel,
  locale,
}: InquiryListProps) {
  const items: InquiryListItem[] = inquiries.map((inq) => ({
    id: inq.id,
    href: `/inquiries/${inq.id}`,
    productName: inq.product_name,
    skuCode: inq.sku_code,
    subject: inq.subject,
    lastMessageAt: new Date(inq.last_message_at).toLocaleString(locale, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    }),
    status: inq.status,
    statusLabel: inq.status === "open" ? openLabel : closedLabel,
    unreadCount: inq.unread_count ?? 0,
  }));

  return (
    <InquiryListPresenter
      title={title}
      description={description}
      emptyLabel={emptyLabel}
      productColumnLabel={productColumnLabel}
      lastMessageColumnLabel={lastMessageColumnLabel}
      statusColumnLabel={statusColumnLabel}
      unreadLabel={unreadLabel}
      items={items}
    />
  );
}
