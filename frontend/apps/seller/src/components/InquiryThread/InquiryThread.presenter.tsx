"use client";

import { type ReactNode } from "react";

export interface InquiryThreadMessage {
  id: string;
  /** True if this message was sent by the seller (the viewer). */
  mine: boolean;
  senderLabel: string;
  body: string;
  timestamp: string;
}

export interface InquiryThreadPresenterProps {
  subject: string;
  productName: string;
  skuCode: string;
  buyerLabel: string;
  statusLabel: string;
  statusTone: "open" | "closed";
  threadClosedNotice?: string;
  messages: InquiryThreadMessage[];
  backLinkSlot?: ReactNode;
  replyFormSlot?: ReactNode;
  closeButtonSlot?: ReactNode;
}

export function InquiryThreadPresenter({
  subject,
  productName,
  skuCode,
  buyerLabel,
  statusLabel,
  statusTone,
  threadClosedNotice,
  messages,
  backLinkSlot,
  replyFormSlot,
  closeButtonSlot,
}: InquiryThreadPresenterProps) {
  return (
    <section className="space-y-4">
      {backLinkSlot}

      <header className="rounded-lg border border-border bg-white p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-xl font-bold text-text-primary">{subject}</h2>
            <p className="mt-1 text-sm text-text-secondary">
              {productName}
              <span className="ml-2 font-mono text-xs text-text-muted">{skuCode}</span>
            </p>
            <p className="mt-0.5 text-xs text-text-secondary">{buyerLabel}</p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                statusTone === "open"
                  ? "bg-green-100 text-green-800"
                  : "bg-gray-100 text-gray-600"
              }`}
            >
              {statusLabel}
            </span>
            {statusTone === "open" && closeButtonSlot}
          </div>
        </div>
      </header>

      <div className="space-y-3" role="log" aria-live="polite">
        {messages.map((msg) => (
          <div
            key={msg.id}
            className={`flex ${msg.mine ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-[80%] rounded-2xl px-4 py-2.5 shadow-sm ${
                msg.mine
                  ? "bg-accent text-white"
                  : "bg-white text-text-primary border border-border"
              }`}
            >
              <div
                className={`text-xs mb-1 ${
                  msg.mine ? "text-white/80" : "text-text-secondary"
                }`}
              >
                <span className="font-semibold">{msg.senderLabel}</span>
                <span className="ml-2">{msg.timestamp}</span>
              </div>
              <p className="whitespace-pre-wrap break-words text-sm">{msg.body}</p>
            </div>
          </div>
        ))}
      </div>

      {statusTone === "closed" && threadClosedNotice ? (
        <div
          className="rounded-md border border-border bg-surface px-4 py-3 text-center text-sm text-text-secondary"
          role="status"
        >
          {threadClosedNotice}
        </div>
      ) : (
        replyFormSlot
      )}
    </section>
  );
}
