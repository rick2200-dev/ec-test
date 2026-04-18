"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import { getMySubscription, listSellerPlans, subscribeToPlan } from "@/lib/api";
import type { SellerSubscription, SubscriptionPlan } from "@/lib/types";
import { PlanCardPresenter } from "./PlanCard.presenter";

export default function SubscriptionPage() {
  const t = useTranslations("subscription");

  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [subscription, setSubscription] = useState<SellerSubscription | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [pendingPlanId, setPendingPlanId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [planList, current] = await Promise.all([
          listSellerPlans(),
          getMySubscription().catch(() => null),
        ]);
        if (cancelled) return;
        setPlans(planList);
        setSubscription(current);
      } catch (err) {
        if (cancelled) return;
        setLoadError(err instanceof Error ? err.message : t("errors.loadFailed"));
      } finally {
        if (!cancelled) setLoaded(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [t]);

  const handleChange = async (plan: SubscriptionPlan) => {
    if (pendingPlanId) return;
    if (typeof window !== "undefined" && !window.confirm(t("confirm", { name: plan.name }))) {
      return;
    }
    setPendingPlanId(plan.id);
    setActionError(null);
    setActionSuccess(null);
    try {
      const next = await subscribeToPlan({ plan_id: plan.id });
      setSubscription(next);
      setActionSuccess(t("actionSuccess", { name: plan.name }));
    } catch (err) {
      if (err instanceof ApiError) {
        setActionError(err.message || t("errors.subscribeFailed"));
      } else {
        setActionError(t("errors.subscribeFailed"));
      }
    } finally {
      setPendingPlanId(null);
    }
  };

  const currentTier = plans.find((p) => p.id === subscription?.plan_id)?.tier ?? -1;

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("title")}</h2>
        <p className="mt-1 text-text-secondary">{t("description")}</p>
      </div>

      {loadError && (
        <div
          role="alert"
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger"
        >
          {loadError}
        </div>
      )}
      {actionError && (
        <div
          role="alert"
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger"
        >
          {actionError}
        </div>
      )}
      {actionSuccess && (
        <div
          role="status"
          className="rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800"
        >
          {actionSuccess}
        </div>
      )}

      {!loaded ? (
        <p className="py-8 text-center text-sm text-text-secondary">{t("loading")}</p>
      ) : plans.length === 0 ? (
        <p className="rounded-lg border border-dashed border-border py-8 text-center text-sm text-text-secondary">
          {t("empty")}
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
          {plans.map((plan) => {
            const isCurrent = plan.id === subscription?.plan_id;
            const isUpgrade = currentTier >= 0 ? plan.tier > currentTier : plan.price_amount > 0;
            return (
              <PlanCardPresenter
                key={plan.id}
                plan={plan}
                isCurrent={isCurrent}
                actionLabel={isUpgrade ? t("upgrade") : t("downgrade")}
                actionVariant={isUpgrade ? "primary" : "secondary"}
                actionDisabled={pendingPlanId !== null}
                onAction={() => handleChange(plan)}
                labels={{
                  maxProducts: t("maxProducts"),
                  searchBoost: t("searchBoost"),
                  featuredSlots: t("featuredSlots"),
                  promotedResults: t("promotedResults"),
                  perMonth: t("perMonth"),
                  free: t("free"),
                  unlimited: t("unlimited"),
                  current: t("current"),
                }}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}
