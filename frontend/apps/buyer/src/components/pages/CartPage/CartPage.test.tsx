import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CartPagePresenter, type CartPagePresenterProps } from "./CartPage.presenter";

const baseProps: CartPagePresenterProps = {
  title: "カート",
  emptyTitle: "カートは空です",
  emptyDescription: "商品を追加してください",
  browseCta: "商品を探す",
  itemColumnLabel: "商品",
  qtyColumnLabel: "数量",
  unitPriceColumnLabel: "単価",
  lineTotalColumnLabel: "小計",
  totalLabel: "合計",
  checkoutCta: "チェックアウト",
  loadingLabel: "読み込み中...",
  loginRequiredTitle: "ログインが必要です",
  loginRequiredCta: "ログイン",
  items: null,
  needsLogin: false,
};

describe("CartPagePresenter", () => {
  it("renders the loading state when items is null", () => {
    render(<CartPagePresenter {...baseProps} />);
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
  });

  it("renders the empty state when items is an empty array", () => {
    render(<CartPagePresenter {...baseProps} items={[]} />);
    expect(screen.getByText("カートは空です")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "商品を探す" })).toBeInTheDocument();
  });

  it("renders the login-required state when needsLogin is true", () => {
    render(<CartPagePresenter {...baseProps} needsLogin items={[]} />);
    const link = screen.getByRole("link", { name: "ログイン" });
    expect(link).toHaveAttribute("href", "/auth/login?returnTo=%2Fcart");
  });

  it("renders cart lines and a computed total", () => {
    render(
      <CartPagePresenter
        {...baseProps}
        items={[
          {
            sku_id: "11111111-1111-1111-1111-111111111111",
            seller_id: "22222222-2222-2222-2222-222222222222",
            quantity: 2,
            unit_price_snapshot: 1500,
            currency: "JPY",
            product_name_snapshot: "テスト商品",
            sku_code_snapshot: "SKU-1",
            added_at: "2026-04-18T00:00:00Z",
          },
          {
            sku_id: "33333333-3333-3333-3333-333333333333",
            seller_id: "22222222-2222-2222-2222-222222222222",
            quantity: 1,
            unit_price_snapshot: 980,
            currency: "JPY",
            product_name_snapshot: "別の商品",
            sku_code_snapshot: "SKU-2",
            added_at: "2026-04-18T00:00:00Z",
          },
        ]}
      />
    );
    expect(screen.getByText("テスト商品")).toBeInTheDocument();
    expect(screen.getByText("別の商品")).toBeInTheDocument();
    // 1500 * 2 + 980 = 3980
    expect(screen.getByText("¥3,980")).toBeInTheDocument();
  });
});
