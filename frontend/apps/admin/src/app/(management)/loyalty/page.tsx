"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import {
  adjustAdminBuyerPoints,
  getAdminBuyerBalance,
  listAdminBuyerTransactions,
} from "@/lib/api";
import type {
  PointsBalance,
  PointsTransaction,
  PointsTransactionType,
} from "@ec-marketplace/types";

export default function LoyaltyAdminPage() {
  const t = useTranslations("loyalty");

  const [buyerInput, setBuyerInput] = useState("");
  const [lookingUp, setLookingUp] = useState(false);
  const [lookupError, setLookupError] = useState<string | null>(null);

  const [buyerId, setBuyerId] = useState<string | null>(null);
  const [balance, setBalance] = useState<PointsBalance | null>(null);
  const [transactions, setTransactions] = useState<PointsTransaction[]>([]);

  const [amount, setAmount] = useState(0);
  const [reason, setReason] = useState("");
  const [adjusting, setAdjusting] = useState(false);
  const [adjustError, setAdjustError] = useState<string | null>(null);
  const [adjustSuccess, setAdjustSuccess] = useState<string | null>(null);

  const lookup = async (id: string) => {
    setLookingUp(true);
    setLookupError(null);
    setAdjustError(null);
    setAdjustSuccess(null);
    try {
      const [bal, tx] = await Promise.all([
        getAdminBuyerBalance(id),
        listAdminBuyerTransactions(id, { limit: 50 }).catch(() => ({
          transactions: [] as PointsTransaction[],
          total: 0,
        })),
      ]);
      setBalance(bal);
      setTransactions(tx.transactions ?? []);
      setBuyerId(id);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setLookupError(t("errors.notFound"));
      } else if (err instanceof Error) {
        setLookupError(err.message);
      } else {
        setLookupError(t("errors.lookupFailed"));
      }
      setBalance(null);
      setTransactions([]);
      setBuyerId(null);
    } finally {
      setLookingUp(false);
    }
  };

  const handleLookup = (e: React.FormEvent) => {
    e.preventDefault();
    const id = buyerInput.trim();
    if (!id) return;
    lookup(id);
  };

  const handleAdjust = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!buyerId || adjusting) return;
    if (!reason.trim()) {
      setAdjustError(t("errors.reasonRequired"));
      return;
    }
    if (amount === 0) {
      setAdjustError(t("errors.amountRequired"));
      return;
    }
    setAdjusting(true);
    setAdjustError(null);
    setAdjustSuccess(null);
    try {
      await adjustAdminBuyerPoints(buyerId, { amount, reason: reason.trim() });
      setAdjustSuccess(t("success"));
      setAmount(0);
      setReason("");
      // Refresh balance + history to reflect the new entry.
      await lookup(buyerId);
    } catch (err) {
      if (err instanceof ApiError) {
        setAdjustError(err.message || t("errors.adjustFailed"));
      } else {
        setAdjustError(t("errors.adjustFailed"));
      }
    } finally {
      setAdjusting(false);
    }
  };

  const txTypeLabels: Record<PointsTransactionType, string> = {
    earn: t("txType.earn"),
    redeem: t("txType.redeem"),
    refund: t("txType.refund"),
    reverse_earn: t("txType.reverseEarn"),
    adjust: t("txType.adjust"),
    expire: t("txType.expire"),
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("title")}</h2>
        <p className="mt-1 text-text-secondary">{t("description")}</p>
      </div>

      <form
        onSubmit={handleLookup}
        className="flex items-end gap-3 rounded-lg border border-border bg-white p-4 shadow-sm"
      >
        <div className="flex-1">
          <label htmlFor="buyer-id" className="block text-sm font-medium text-text-primary">
            {t("lookup.label")}
          </label>
          <input
            id="buyer-id"
            type="text"
            value={buyerInput}
            onChange={(e) => setBuyerInput(e.target.value)}
            placeholder="auth0|..."
            className="mt-1 w-full rounded-md border border-border px-3 py-2 font-mono text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>
        <button
          type="submit"
          disabled={lookingUp || buyerInput.trim().length === 0}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
        >
          {lookingUp ? t("lookup.searching") : t("lookup.search")}
        </button>
      </form>

      {lookupError && (
        <div
          role="alert"
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger"
        >
          {lookupError}
        </div>
      )}

      {balance && buyerId && (
        <section className="rounded-lg border border-border bg-white p-6 shadow-sm">
          <h3 className="text-base font-semibold text-text-primary">{t("balance.title")}</h3>
          <p className="mt-1 text-xs text-text-secondary">
            {t("balance.buyerId")}: <span className="font-mono">{buyerId}</span>
          </p>
          <dl className="mt-4 grid grid-cols-4 gap-4 text-sm">
            <div>
              <dt className="text-text-secondary">{t("balance.spendable")}</dt>
              <dd className="mt-1 text-2xl font-bold text-accent">
                {balance.spendable.toLocaleString()} pt
              </dd>
            </div>
            <div>
              <dt className="text-text-secondary">{t("balance.pending")}</dt>
              <dd className="mt-1 text-text-primary">
                {balance.pending_redemption.toLocaleString()}
              </dd>
            </div>
            <div>
              <dt className="text-text-secondary">{t("balance.lifetimeEarned")}</dt>
              <dd className="mt-1 text-text-primary">{balance.lifetime_earned.toLocaleString()}</dd>
            </div>
            <div>
              <dt className="text-text-secondary">{t("balance.lifetimeRedeemed")}</dt>
              <dd className="mt-1 text-text-primary">
                {balance.lifetime_redeemed.toLocaleString()}
              </dd>
            </div>
          </dl>
        </section>
      )}

      {buyerId && (
        <section className="rounded-lg border border-border bg-white p-6 shadow-sm">
          <h3 className="text-base font-semibold text-text-primary">{t("adjust.title")}</h3>
          <p className="mt-1 text-xs text-text-secondary">{t("adjust.hint")}</p>

          {adjustError && (
            <div
              role="alert"
              className="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger"
            >
              {adjustError}
            </div>
          )}
          {adjustSuccess && (
            <div
              role="status"
              className="mt-3 rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-800"
            >
              {adjustSuccess}
            </div>
          )}

          <form onSubmit={handleAdjust} className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label className="mb-1 block text-sm font-medium text-text-primary">
                {t("adjust.amountLabel")} <span className="text-danger">*</span>
              </label>
              <input
                type="number"
                value={amount}
                onChange={(e) => setAmount(Number(e.target.value))}
                className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
              <p className="mt-1 text-xs text-text-secondary">{t("adjust.amountHint")}</p>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-text-primary">
                {t("adjust.reasonLabel")} <span className="text-danger">*</span>
              </label>
              <input
                type="text"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder={t("adjust.reasonPlaceholder")}
                className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>
            <div className="md:col-span-2">
              <button
                type="submit"
                disabled={adjusting}
                className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-hover disabled:opacity-50"
              >
                {adjusting ? t("adjust.submitting") : t("adjust.submit")}
              </button>
            </div>
          </form>
        </section>
      )}

      {buyerId && (
        <section className="rounded-lg border border-border bg-white shadow-sm">
          <h3 className="border-b border-border p-4 text-base font-semibold text-text-primary">
            {t("history.title")}
          </h3>
          {transactions.length === 0 ? (
            <p className="p-6 text-center text-sm text-text-secondary">{t("history.empty")}</p>
          ) : (
            <table className="min-w-full divide-y divide-border text-sm">
              <thead className="bg-surface text-left text-xs font-medium uppercase tracking-wider text-text-secondary">
                <tr>
                  <th className="px-4 py-3">{t("history.date")}</th>
                  <th className="px-4 py-3">{t("history.type")}</th>
                  <th className="px-4 py-3 text-right">{t("history.amount")}</th>
                  <th className="px-4 py-3 text-right">{t("history.balance")}</th>
                  <th className="px-4 py-3">{t("history.note")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {transactions.map((tx) => (
                  <tr key={tx.id}>
                    <td className="px-4 py-3 text-text-secondary">
                      {new Date(tx.created_at).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-text-primary">
                      {txTypeLabels[tx.type] ?? tx.type}
                    </td>
                    <td
                      className={`px-4 py-3 text-right font-semibold ${
                        tx.amount >= 0 ? "text-green-700" : "text-red-700"
                      }`}
                    >
                      {tx.amount >= 0 ? "+" : ""}
                      {tx.amount.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right text-text-primary">
                      {tx.balance_after.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-text-secondary">{tx.note ?? ""}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      )}
    </div>
  );
}
