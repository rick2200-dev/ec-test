"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { markBuyerInquiryRead, postBuyerInquiryMessage } from "@/lib/api";
import type { InquiryMessage, InquiryWithMessages } from "@/lib/types";
import {
  InquiryThreadPresenter,
  type InquiryThreadMessage,
} from "./InquiryThread.presenter";

export interface InquiryThreadProps {
  /** Initial thread data fetched server-side. */
  initial: InquiryWithMessages;
  /** Back-navigation href (buyer app → /inquiries). */
  backHref: string;
  locale: string;
}

export default function InquiryThread({ initial, backHref, locale }: InquiryThreadProps) {
  const t = useTranslations("inquiries");
  const [thread, setThread] = useState<InquiryWithMessages>(initial);
  const [body, setBody] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Mark any unread seller messages as read on mount.
  useEffect(() => {
    const hasUnread = thread.messages.some(
      (m) => m.sender_type === "seller" && !m.read_at,
    );
    if (!hasUnread) return;
    void markBuyerInquiryRead(thread.id).catch(() => {
      // best-effort; not fatal for the UI
    });
  }, [thread.id, thread.messages]);

  const onSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!body.trim() || sending) return;
    setSending(true);
    setError(null);
    try {
      const created: InquiryMessage = await postBuyerInquiryMessage(thread.id, body.trim());
      setThread((prev) => ({
        ...prev,
        last_message_at: created.created_at,
        messages: [...prev.messages, created],
      }));
      setBody("");
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setSending(false);
    }
  };

  const messages: InquiryThreadMessage[] = thread.messages.map((m) => ({
    id: m.id,
    mine: m.sender_type === "buyer",
    senderLabel: m.sender_type === "buyer" ? t("sender.you") : t("sender.seller"),
    body: m.body,
    timestamp: new Date(m.created_at).toLocaleString(locale, {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    }),
  }));

  const closed = thread.status === "closed";

  return (
    <InquiryThreadPresenter
      subject={thread.subject}
      productName={thread.product_name}
      skuCode={thread.sku_code}
      statusLabel={closed ? t("status.closed") : t("status.open")}
      statusTone={thread.status}
      threadClosedNotice={t("threadClosed")}
      messages={messages}
      backLinkSlot={
        <Link
          href={backHref}
          className="inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
        >
          ← {t("backToList")}
        </Link>
      }
      replyFormSlot={
        <form
          onSubmit={onSubmit}
          className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm"
        >
          <label className="block">
            <span className="text-sm font-medium text-gray-700">{t("body")}</span>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={4}
              maxLength={4000}
              placeholder={t("bodyPlaceholder")}
              className="mt-1 w-full resize-none rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              required
            />
          </label>
          {error && (
            <p className="mt-2 text-sm text-red-600" role="alert">
              {error}
            </p>
          )}
          <div className="mt-3 flex justify-end">
            <button
              type="submit"
              disabled={sending || !body.trim()}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300"
            >
              {sending ? t("sending") : t("send")}
            </button>
          </div>
        </form>
      }
    />
  );
}
