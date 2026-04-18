import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { SubscriptionPlan } from "@/lib/types";
import { PlanCardPresenter } from "./PlanCard.presenter";

const labels = {
  maxProducts: "最大商品数",
  searchBoost: "検索ブースト",
  featuredSlots: "注目枠数",
  promotedResults: "プロモーション数",
  perMonth: "/月",
  free: "無料",
  unlimited: "無制限",
  current: "現在のプラン",
};

const plan: SubscriptionPlan = {
  id: "plan-1",
  name: "Standard",
  slug: "standard",
  tier: 1,
  price_amount: 9800,
  price_currency: "JPY",
  features: { max_products: 50, search_boost: 1.5, featured_slots: 2, promoted_results: 0 },
};

describe("PlanCardPresenter", () => {
  it("renders the current-plan pill when isCurrent is true", () => {
    render(
      <PlanCardPresenter
        plan={plan}
        isCurrent
        actionLabel="(unused)"
        actionDisabled
        actionVariant="secondary"
        onAction={() => {}}
        labels={labels}
      />
    );
    expect(screen.getByText("現在のプラン")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("shows an action button and emits onAction", async () => {
    const onAction = vi.fn();
    render(
      <PlanCardPresenter
        plan={plan}
        isCurrent={false}
        actionLabel="アップグレード"
        actionDisabled={false}
        actionVariant="primary"
        onAction={onAction}
        labels={labels}
      />
    );
    await userEvent.click(screen.getByRole("button", { name: "アップグレード" }));
    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it("renders 無制限 when max_products < 0", () => {
    render(
      <PlanCardPresenter
        plan={{ ...plan, features: { ...plan.features, max_products: -1 } }}
        isCurrent={false}
        actionLabel="選ぶ"
        actionDisabled={false}
        actionVariant="primary"
        onAction={() => {}}
        labels={labels}
      />
    );
    expect(screen.getByText("無制限")).toBeInTheDocument();
  });
});
