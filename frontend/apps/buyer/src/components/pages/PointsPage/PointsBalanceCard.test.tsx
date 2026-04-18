import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PointsBalanceCardPresenter } from "./PointsBalanceCard.presenter";

const labels = {
  spendable: "利用可能ポイント",
  pending: "保留中",
  lifetimeEarned: "累計獲得",
  lifetimeRedeemed: "累計利用",
  pointsUnit: "pt",
};

describe("PointsBalanceCardPresenter", () => {
  it("renders the spendable balance prominently", () => {
    render(
      <PointsBalanceCardPresenter
        spendable={1234}
        pendingRedemption={50}
        lifetimeEarned={5000}
        lifetimeRedeemed={3766}
        labels={labels}
      />
    );
    expect(screen.getByText("1,234")).toBeInTheDocument();
    expect(screen.getByText("利用可能ポイント")).toBeInTheDocument();
    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("5,000")).toBeInTheDocument();
    expect(screen.getByText("3,766")).toBeInTheDocument();
  });
});
