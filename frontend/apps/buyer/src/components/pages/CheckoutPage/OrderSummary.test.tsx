import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { OrderSummaryPresenter } from "./OrderSummary.presenter";

const labels = {
  title: "ご注文内容",
  items: "商品数",
  subtotal: "小計",
  couponDiscount: "クーポン割引",
  pointsRedeemed: "ポイント利用",
  total: "合計",
};

describe("OrderSummaryPresenter", () => {
  it("renders subtotal and total without optional rows", () => {
    render(
      <OrderSummaryPresenter
        itemsCount={2}
        subtotal={1000}
        couponDiscount={0}
        pointsRedeemed={0}
        total={1000}
        labels={labels}
      />
    );
    // ¥1,000 appears twice: once for subtotal, once for total. We want the
    // discount/points rows hidden — assert by label, not by amount.
    expect(screen.getAllByText("¥1,000")).toHaveLength(2);
    expect(screen.queryByText("クーポン割引")).not.toBeInTheDocument();
    expect(screen.queryByText("ポイント利用")).not.toBeInTheDocument();
  });

  it("shows the coupon discount row when set", () => {
    render(
      <OrderSummaryPresenter
        itemsCount={2}
        subtotal={1000}
        couponDiscount={200}
        pointsRedeemed={0}
        total={800}
        labels={labels}
      />
    );
    expect(screen.getByText("クーポン割引")).toBeInTheDocument();
    expect(screen.getByText("−¥200")).toBeInTheDocument();
    expect(screen.getByText("¥800")).toBeInTheDocument();
  });

  it("shows the points row when redeeming", () => {
    render(
      <OrderSummaryPresenter
        itemsCount={2}
        subtotal={1000}
        couponDiscount={0}
        pointsRedeemed={150}
        total={850}
        labels={labels}
      />
    );
    expect(screen.getByText("ポイント利用")).toBeInTheDocument();
    expect(screen.getByText("−¥150")).toBeInTheDocument();
  });
});
