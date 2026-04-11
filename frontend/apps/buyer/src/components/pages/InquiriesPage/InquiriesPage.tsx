"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { listBuyerInquiries } from "@/lib/api";
import InquiryList from "@/components/InquiryList";
import type { Inquiry } from "@/lib/types";
import { InquiriesPagePresenter } from "./InquiriesPage.presenter";

export default function InquiriesPage() {
  const t = useTranslations("inquiries");
  const locale = useLocale();
  const [items, setItems] = useState<Inquiry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await listBuyerInquiries({ limit: 50 });
        if (!cancelled) setItems(res.items ?? []);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("errorGeneric"));
        }
      } finally {
        if (!cancelled) setLoaded(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [t]);

  return (
    <InquiriesPagePresenter
      errorSlot={
        error ? (
          <div
            className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
            role="alert"
          >
            {error}
          </div>
        ) : undefined
      }
      listSlot={
        <InquiryList
          inquiries={loaded ? items : []}
          title={t("title")}
          description={t("description")}
          emptyLabel={loaded ? t("empty") : "..."}
          productColumnLabel={t("product")}
          lastMessageColumnLabel={t("lastMessageAt")}
          statusColumnLabel={t("status.columnLabel")}
          unreadLabel={t("unread")}
          openLabel={t("status.open")}
          closedLabel={t("status.closed")}
          buildHref={(id) => `/inquiries/${id}`}
          locale={locale}
        />
      }
    />
  );
}
