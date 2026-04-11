"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import {
  closeSellerInquiry,
  markSellerInquiryRead,
  postSellerInquiryMessage,
} from "@/lib/api";
import type { InquiryMessage, InquiryWithMessages } from "@/lib/types";
import {
  InquiryThreadPresenter,
  type InquiryThreadMessage,
} from "./InquiryThread.presenter";

export interface InquiryThreadProps {
  initial: InquiryWithMessages;
  backHref: string;
  locale: string;
}

export default function InquiryThread({ initial, backHref, locale }: InquiryThreadProps) {
  const t = useTranslations("inquiries");
  const [thread, setThread] = useState<InquiryWithMessages>(initial);
  const [body, setBody] = useState("");
  const [sending, setSending] = useState(false);
  const [closing, setClosing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const hasUnread = thread.messages.some(
      (m) => m.sender_type === "buyer" && !m.read_at,
    );
    if (!hasUnread) return;
    void markSellerInquiryRead(thread.id).catch(() => {
      // best-effort
    });
  }, [thread.id, thread.messages]);

  const onSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!body.trim() || sending) return;
    setSending(true);
    setError(null);
    try {
      const created: InquiryMessage = await postSellerInquiryMessage(thread.id, body.trim());
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

  const onClose = async () => {
    if (closing) return;
    setClosing(true);
    setError(null);
    try {
      const updated = await closeSellerInquiry(thread.id);
      setThread((prev) => ({ ...prev, status: updated.status }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    } finally {
      setClosing(false);
    }
  };

  const messages: InquiryThreadMessage[] = thread.messages.map((m) => ({
    id: m.id,
    mine: m.sender_type === "seller",
    senderLabel: m.sender_type === "seller" ? t("sender.you") : t("sender.buyer"),
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
      buyerLabel={`${t("buyer")}: ${thread.buyer_auth0_id}`}
      statusLabel={closed ? t("status.closed") : t("status.open")}
      statusTone={thread.status}
      threadClosedNotice={t("closed")}
      messages={messages}
      backLinkSlot={
        <Link
          href={backHref}
          className="inline-flex items-center text-sm text-accent hover:text-accent-dark"
        >
          ← {t("backToList")}
        </Link>
      }
      closeButtonSlot={
        <button
          type="button"
          onClick={onClose}
          disabled={closing}
          className="rounded-md border border-border bg-white px-3 py-1 text-xs font-medium text-text-primary hover:bg-surface disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t("close")}
        </button>
      }
      replyFormSlot={
        <form
          onSubmit={onSubmit}
          className="rounded-lg border border-border bg-white p-4 shadow-sm"
        >
          <label className="block">
            <span className="text-sm font-medium text-text-primary">{t("body")}</span>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              rows={4}
              maxLength={4000}
              placeholder={t("bodyPlaceholder")}
              required
              className="mt-1 w-full resize-none rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
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
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-white hover:bg-accent-dark disabled:cursor-not-allowed disabled:opacity-50"
            >
              {sending ? t("sending") : t("send")}
            </button>
          </div>
        </form>
      }
    />
  );
}
