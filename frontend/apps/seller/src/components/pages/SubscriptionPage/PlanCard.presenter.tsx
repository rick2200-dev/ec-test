import type { SubscriptionPlan } from "@/lib/types";

export interface PlanCardPresenterProps {
  plan: SubscriptionPlan;
  isCurrent: boolean;
  /** Label rendered inside the action button/pill. */
  actionLabel: string;
  /** Controls whether the action is interactive; pass false for "Current Plan". */
  actionDisabled: boolean;
  onAction: () => void;
  actionVariant: "primary" | "secondary";
  labels: {
    maxProducts: string;
    searchBoost: string;
    featuredSlots: string;
    promotedResults: string;
    perMonth: string;
    free: string;
    unlimited: string;
    current: string;
  };
}

export function PlanCardPresenter({
  plan,
  isCurrent,
  actionLabel,
  actionDisabled,
  onAction,
  actionVariant,
  labels,
}: PlanCardPresenterProps) {
  const priceLabel =
    plan.price_amount === 0 ? labels.free : `\u00A5${plan.price_amount.toLocaleString()}`;
  const maxProductsLabel =
    plan.features.max_products < 0 ? labels.unlimited : String(plan.features.max_products);

  return (
    <div
      className={`flex flex-col rounded-lg border bg-white p-6 shadow-sm ${
        isCurrent ? "border-accent ring-2 ring-accent/20" : "border-border"
      }`}
    >
      <div className="mb-4">
        <h3 className="text-lg font-bold text-text-primary">{plan.name}</h3>
        <p className="mt-2 text-3xl font-bold text-text-primary">
          {priceLabel}
          {plan.price_amount > 0 && (
            <span className="text-sm font-normal text-text-secondary">{labels.perMonth}</span>
          )}
        </p>
      </div>

      <ul className="mb-6 flex-1 space-y-3 text-sm">
        <li className="flex justify-between">
          <span className="text-text-secondary">{labels.maxProducts}</span>
          <span className="font-medium text-text-primary">{maxProductsLabel}</span>
        </li>
        <li className="flex justify-between">
          <span className="text-text-secondary">{labels.searchBoost}</span>
          <span className="font-medium text-text-primary">x{plan.features.search_boost}</span>
        </li>
        <li className="flex justify-between">
          <span className="text-text-secondary">{labels.featuredSlots}</span>
          <span className="font-medium text-text-primary">{plan.features.featured_slots}</span>
        </li>
        <li className="flex justify-between">
          <span className="text-text-secondary">{labels.promotedResults}</span>
          <span className="font-medium text-text-primary">{plan.features.promoted_results}</span>
        </li>
      </ul>

      <div className="text-center">
        {isCurrent ? (
          <span className="inline-block rounded-lg bg-surface px-4 py-2 text-sm font-medium text-text-secondary">
            {labels.current}
          </span>
        ) : (
          <button
            type="button"
            onClick={onAction}
            disabled={actionDisabled}
            className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors disabled:opacity-50 ${
              actionVariant === "primary"
                ? "bg-accent text-white hover:bg-accent-hover"
                : "border border-border bg-surface text-text-primary hover:bg-surface-hover"
            }`}
          >
            {actionLabel}
          </button>
        )}
      </div>
    </div>
  );
}
