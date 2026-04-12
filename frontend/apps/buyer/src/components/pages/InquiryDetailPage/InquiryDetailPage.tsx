"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { getBuyerInquiry } from "@/lib/api";
import InquiryThread from "@/components/InquiryThread";
import type { InquiryWithMessages } from "@/lib/types";
import { InquiryDetailPagePresenter } from "./InquiryDetailPage.presenter";

export interface InquiryDetailPageProps {
  id: string;
}

export default function InquiryDetailPage({ id }: InquiryDetailPageProps) {
  const t = useTranslations("inquiries");
  const locale = useLocale();
  const [thread, setThread] = useState<InquiryWithMessages | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const data = await getBuyerInquiry(id);
        if (!cancelled) setThread(data);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("errorGeneric"));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, t]);

  let threadSlot;
  if (error) {
    threadSlot = (
      <div
        className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
        role="alert"
      >
        {error}
      </div>
    );
  } else if (!thread) {
    threadSlot = <div className="text-sm text-gray-500">...</div>;
  } else {
    threadSlot = <InquiryThread initial={thread} backHref="/inquiries" locale={locale} />;
  }

  return <InquiryDetailPagePresenter threadSlot={threadSlot} />;
}
