"use client";

import { type ReactNode } from "react";

export interface InquiryThreadMessage {
  id: string;
  /** True if the viewer sent this message (right-aligned, accent bubble). */
  mine: boolean;
  senderLabel: string;
  body: string;
  timestamp: string;
}

export interface InquiryThreadPresenterProps {
  subject: string;
  productName: string;
  skuCode: string;
  statusLabel: string;
  statusTone: "open" | "closed";
  threadClosedNotice?: string;
  messages: InquiryThreadMessage[];
  /** Back link content is a rendered element so the container can drop in a Next.js Link. */
  backLinkSlot?: ReactNode;
  /** Reply form — omit when the thread is closed or the viewer cannot reply. */
  replyFormSlot?: ReactNode;
}

export function InquiryThreadPresenter({
  subject,
  productName,
  skuCode,
  statusLabel,
  statusTone,
  threadClosedNotice,
  messages,
  backLinkSlot,
  replyFormSlot,
}: InquiryThreadPresenterProps) {
  return (
    <section className="space-y-4">
      {backLinkSlot}

      <header className="rounded-lg border border-gray-200 bg-white p-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-gray-900">{subject}</h1>
            <p className="mt-1 text-sm text-gray-500">
              {productName}
              <span className="ml-2 font-mono text-xs text-gray-400">{skuCode}</span>
            </p>
          </div>
          <span
            className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
              statusTone === "open"
                ? "bg-green-100 text-green-800"
                : "bg-gray-100 text-gray-600"
            }`}
          >
            {statusLabel}
          </span>
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
                  ? "bg-blue-600 text-white"
                  : "bg-white text-gray-900 border border-gray-200"
              }`}
            >
              <div
                className={`text-xs mb-1 ${
                  msg.mine ? "text-blue-100" : "text-gray-500"
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
          className="rounded-md border border-gray-200 bg-gray-50 px-4 py-3 text-center text-sm text-gray-600"
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
