"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import { adminRefundOrder, type AdminRefundResponse } from "@/lib/api";

// UUID v4 sanity check so we catch typos before hitting the backend
// with a value the router will return 400 for anyway. Not a strict
// validation (v1-v5 all match) — just a shape guard.
const UUID_REGEX = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export default function AdminRefundsPage() {
  const t = useTranslations("refunds");
  const [orderId, setOrderId] = useState("");
  const [reason, setReason] = useState("");
  const [confirmArmed, setConfirmArmed] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<AdminRefundResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  const canSubmit =
    !submitting && UUID_REGEX.test(orderId.trim()) && reason.trim().length > 0;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    if (!confirmArmed) {
      setConfirmArmed(true);
      return;
    }
    setSubmitting(true);
    setError(null);
    setResult(null);
    try {
      const res = await adminRefundOrder(orderId.trim(), reason.trim());
      setResult(res);
      setOrderId("");
      setReason("");
      setConfirmArmed(false);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || t("errors.generic"));
      } else {
        setError(t("errors.generic"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("title")}</h2>
        <p className="mt-1 text-text-secondary">{t("description")}</p>
      </div>

      <div
        role="note"
        className="rounded-md border border-yellow-300 bg-yellow-50 px-4 py-3 text-sm text-yellow-900"
      >
        {t("warning")}
      </div>

      {error && (
        <div
          role="alert"
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger"
        >
          {error}
        </div>
      )}

      {result && (
        <div
          role="status"
          className="rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-900"
        >
          <p className="font-medium">{t("success.title")}</p>
          <ul className="mt-2 space-y-1">
            <li>
              {t("success.requestId")}: <span className="font-mono">{result.id}</span>
            </li>
            <li>
              {t("success.orderId")}: <span className="font-mono">{result.order_id}</span>
            </li>
            {result.stripe_refund_id && (
              <li>
                {t("success.stripeRefund")}:{" "}
                <span className="font-mono">{result.stripe_refund_id}</span>
              </li>
            )}
          </ul>
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="space-y-4 rounded-lg border border-border bg-white p-6 shadow-sm"
      >
        <div>
          <label className="mb-1 block text-sm font-medium text-text-primary">
            {t("fields.orderId")} <span className="text-danger">*</span>
          </label>
          <input
            type="text"
            value={orderId}
            onChange={(e) => {
              setOrderId(e.target.value);
              setConfirmArmed(false);
            }}
            placeholder="00000000-0000-0000-0000-000000000000"
            required
            className="w-full rounded-md border border-border px-3 py-2 font-mono text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
          <p className="mt-1 text-xs text-text-secondary">{t("fields.orderIdHint")}</p>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-text-primary">
            {t("fields.reason")} <span className="text-danger">*</span>
          </label>
          <textarea
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setConfirmArmed(false);
            }}
            rows={3}
            required
            className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            placeholder={t("fields.reasonPlaceholder")}
          />
          <p className="mt-1 text-xs text-text-secondary">{t("fields.reasonHint")}</p>
        </div>

        <div className="flex items-center gap-3">
          <button
            type="submit"
            disabled={!canSubmit}
            className={`rounded-lg px-4 py-2 text-sm font-medium text-white transition-colors disabled:opacity-50 ${
              confirmArmed
                ? "bg-red-600 hover:bg-red-700"
                : "bg-accent hover:bg-accent-hover"
            }`}
          >
            {submitting
              ? t("buttons.submitting")
              : confirmArmed
              ? t("buttons.confirm")
              : t("buttons.submit")}
          </button>
          {confirmArmed && !submitting && (
            <button
              type="button"
              onClick={() => setConfirmArmed(false)}
              className="rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-primary hover:bg-surface-hover"
            >
              {t("buttons.cancel")}
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
