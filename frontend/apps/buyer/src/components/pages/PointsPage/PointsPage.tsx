"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import {
  ApiError,
  getPointsBalance,
  listMyCouponRedemptions,
  listPointsTransactions,
} from "@/lib/api";
import type {
  CouponRedemption,
  PointsBalance,
  PointsTransaction,
  PointsTransactionType,
} from "@ec-marketplace/types";
import { PointsPagePresenter, type PointsPageTab } from "./PointsPage.presenter";
import { PointsBalanceCardPresenter } from "./PointsBalanceCard.presenter";
import { PointsTransactionsListPresenter } from "./PointsTransactionsList.presenter";
import { CouponRedemptionsListPresenter } from "./CouponRedemptionsList.presenter";

const PAGE_SIZE = 50;

export default function PointsPage() {
  const t = useTranslations("points");
  const locale = useLocale();

  const [balance, setBalance] = useState<PointsBalance | null>(null);
  const [needsLogin, setNeedsLogin] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [activeTab, setActiveTab] = useState<PointsPageTab>("transactions");

  const [transactions, setTransactions] = useState<PointsTransaction[]>([]);
  const [txLoaded, setTxLoaded] = useState(false);

  const [redemptions, setRedemptions] = useState<CouponRedemption[]>([]);
  const [rdLoaded, setRdLoaded] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const b = await getPointsBalance();
        if (!cancelled) setBalance(b);
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) {
          setNeedsLogin(true);
          return;
        }
        setError(err instanceof Error ? err.message : t("errorGeneric"));
      }
      try {
        const tx = await listPointsTransactions({ limit: PAGE_SIZE });
        if (!cancelled) setTransactions(tx.transactions ?? []);
      } catch {
        // Surface tx errors quietly — the empty state is acceptable.
      } finally {
        if (!cancelled) setTxLoaded(true);
      }
      try {
        const rd = await listMyCouponRedemptions({ limit: PAGE_SIZE });
        if (!cancelled) setRedemptions(rd.redemptions ?? []);
      } catch {
        // Same as above.
      } finally {
        if (!cancelled) setRdLoaded(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [t]);

  if (needsLogin) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-12 text-center">
        <h1 className="text-2xl font-bold text-gray-900">{t("loginRequired.title")}</h1>
        <a
          href="/auth/login?returnTo=%2Fpoints"
          className="mt-6 inline-block rounded-md bg-blue-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-blue-700"
        >
          {t("loginRequired.cta")}
        </a>
      </div>
    );
  }

  const txTypeLabels: Record<PointsTransactionType, string> = {
    earn: t("txType.earn"),
    redeem: t("txType.redeem"),
    refund: t("txType.refund"),
    reverse_earn: t("txType.reverseEarn"),
    adjust: t("txType.adjust"),
    expire: t("txType.expire"),
  };

  return (
    <PointsPagePresenter
      title={t("title")}
      tabs={[
        { id: "transactions", label: t("tabs.transactions") },
        { id: "coupons", label: t("tabs.coupons") },
      ]}
      activeTab={activeTab}
      onTabChange={setActiveTab}
      balanceSlot={
        <PointsBalanceCardPresenter
          spendable={balance?.spendable ?? 0}
          pendingRedemption={balance?.pending_redemption ?? 0}
          lifetimeEarned={balance?.lifetime_earned ?? 0}
          lifetimeRedeemed={balance?.lifetime_redeemed ?? 0}
          labels={{
            spendable: t("balance.spendable"),
            pending: t("balance.pending"),
            lifetimeEarned: t("balance.lifetimeEarned"),
            lifetimeRedeemed: t("balance.lifetimeRedeemed"),
            pointsUnit: t("balance.pointsUnit"),
          }}
        />
      }
      contentSlot={
        activeTab === "transactions" ? (
          <PointsTransactionsListPresenter
            transactions={transactions}
            emptyLabel={t("transactions.empty")}
            loadingLabel={t("loading")}
            loaded={txLoaded}
            typeLabels={txTypeLabels}
            columns={{
              date: t("transactions.columns.date"),
              type: t("transactions.columns.type"),
              amount: t("transactions.columns.amount"),
              balanceAfter: t("transactions.columns.balanceAfter"),
              note: t("transactions.columns.note"),
            }}
            locale={locale}
          />
        ) : (
          <CouponRedemptionsListPresenter
            redemptions={redemptions}
            emptyLabel={t("coupons.empty")}
            loadingLabel={t("loading")}
            loaded={rdLoaded}
            columns={{
              date: t("coupons.columns.date"),
              orderId: t("coupons.columns.orderId"),
              discount: t("coupons.columns.discount"),
              refunded: t("coupons.columns.refunded"),
              status: t("coupons.columns.status"),
            }}
            statusLabels={{
              active: t("coupons.status.active"),
              refundedFully: t("coupons.status.refundedFully"),
              refundedPartially: t("coupons.status.refundedPartially"),
            }}
            locale={locale}
          />
        )
      }
      errorSlot={
        error ? (
          <div
            role="alert"
            className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          >
            {error}
          </div>
        ) : undefined
      }
    />
  );
}
