"use client";

import { useTranslations } from "next-intl";
import type { SubscriptionPlan } from "@/lib/types";

// Mock data for available plans - will be fetched from API in production.
const plans: SubscriptionPlan[] = [
  {
    id: "plan-001",
    name: "Free",
    slug: "free",
    tier: 0,
    price_amount: 0,
    price_currency: "JPY",
    features: { max_products: 10, search_boost: 1.0, featured_slots: 0, promoted_results: 0 },
  },
  {
    id: "plan-002",
    name: "Standard",
    slug: "standard",
    tier: 1,
    price_amount: 9800,
    price_currency: "JPY",
    features: { max_products: 50, search_boost: 1.5, featured_slots: 2, promoted_results: 0 },
  },
  {
    id: "plan-003",
    name: "Premium",
    slug: "premium",
    tier: 2,
    price_amount: 29800,
    price_currency: "JPY",
    features: { max_products: -1, search_boost: 2.5, featured_slots: 5, promoted_results: 3 },
  },
];

// Mock current plan
const currentPlanSlug = "free";

export default function SubscriptionPage() {
  const t = useTranslations();

  const formatPrice = (amount: number) => {
    if (amount === 0) return t("subscription.free");
    return `\u00A5${amount.toLocaleString()}`;
  };

  const formatMaxProducts = (count: number) => {
    if (count < 0) return t("subscription.unlimited");
    return count.toString();
  };

  const getPlanAction = (plan: SubscriptionPlan) => {
    if (plan.slug === currentPlanSlug) {
      return (
        <span className="inline-block px-4 py-2 bg-gray-100 text-gray-500 rounded-lg text-sm font-medium">
          {t("subscription.current")}
        </span>
      );
    }
    const currentPlan = plans.find((p) => p.slug === currentPlanSlug);
    const isUpgrade = currentPlan ? plan.tier > currentPlan.tier : false;
    return (
      <button
        className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
          isUpgrade
            ? "bg-accent text-white hover:bg-accent-hover"
            : "bg-surface text-text-primary border border-border hover:bg-surface-hover"
        }`}
        onClick={() => {
          // TODO: API call to POST /api/v1/seller/subscription
          alert(`Plan change to ${plan.name} (mock)`);
        }}
      >
        {isUpgrade ? t("subscription.upgrade") : t("subscription.downgrade")}
      </button>
    );
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("subscription.title")}</h2>
        <p className="text-text-secondary mt-1">{t("subscription.description")}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {plans.map((plan) => {
          const isCurrent = plan.slug === currentPlanSlug;
          return (
            <div
              key={plan.id}
              className={`bg-white rounded-lg border shadow-sm p-6 flex flex-col ${
                isCurrent ? "border-accent ring-2 ring-accent/20" : "border-border"
              }`}
            >
              <div className="mb-4">
                <h3 className="text-lg font-bold text-text-primary">{plan.name}</h3>
                <p className="text-3xl font-bold text-text-primary mt-2">
                  {formatPrice(plan.price_amount)}
                  {plan.price_amount > 0 && (
                    <span className="text-sm font-normal text-text-secondary">
                      {t("subscription.perMonth")}
                    </span>
                  )}
                </p>
              </div>

              <ul className="space-y-3 mb-6 flex-1">
                <li className="flex justify-between text-sm">
                  <span className="text-text-secondary">{t("subscription.maxProducts")}</span>
                  <span className="font-medium text-text-primary">
                    {formatMaxProducts(plan.features.max_products)}
                  </span>
                </li>
                <li className="flex justify-between text-sm">
                  <span className="text-text-secondary">{t("subscription.searchBoost")}</span>
                  <span className="font-medium text-text-primary">x{plan.features.search_boost}</span>
                </li>
                <li className="flex justify-between text-sm">
                  <span className="text-text-secondary">{t("subscription.featuredSlots")}</span>
                  <span className="font-medium text-text-primary">{plan.features.featured_slots}</span>
                </li>
                <li className="flex justify-between text-sm">
                  <span className="text-text-secondary">{t("subscription.promotedResults")}</span>
                  <span className="font-medium text-text-primary">{plan.features.promoted_results}</span>
                </li>
              </ul>

              <div className="text-center">{getPlanAction(plan)}</div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
