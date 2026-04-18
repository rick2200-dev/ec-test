import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SearchResultsGridPresenter } from "./SearchResultsGrid.presenter";

describe("SearchResultsGridPresenter", () => {
  it("renders the empty state when no results", () => {
    render(<SearchResultsGridPresenter results={[]} emptyLabel="該当なし" />);
    expect(screen.getByText("該当なし")).toBeInTheDocument();
  });

  it("renders product hits as links", () => {
    render(
      <SearchResultsGridPresenter
        emptyLabel=""
        results={[
          {
            id: "p1",
            seller_id: "s1",
            name: "テスト商品",
            slug: "test-product",
            description: "",
            status: "active",
            price_amount: 1500,
            price_currency: "JPY",
            seller_name: "テストショップ",
            category_name: "雑貨",
            score: 1,
            is_promoted: false,
            plan_tier: 0,
          },
        ]}
      />
    );
    const link = screen.getByRole("link", { name: /テスト商品/ });
    expect(link).toHaveAttribute("href", "/products/test-product");
    expect(screen.getByText("テストショップ")).toBeInTheDocument();
    expect(screen.getByText("¥1,500")).toBeInTheDocument();
  });
});
