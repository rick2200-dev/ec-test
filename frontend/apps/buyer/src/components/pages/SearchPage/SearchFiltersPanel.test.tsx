import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SearchFiltersPanelPresenter } from "./SearchFiltersPanel.presenter";

const labels = {
  categories: "カテゴリ",
  priceRange: "価格帯",
  clearFilters: "条件をクリア",
};

const priceRangeLabels = {
  under_1000: "< ¥1,000",
  "1000_5000": "¥1,000 - ¥5,000",
};

describe("SearchFiltersPanelPresenter", () => {
  it("renders category and price facets", () => {
    render(
      <SearchFiltersPanelPresenter
        facets={[
          {
            field: "category",
            values: [
              { value: "Electronics", count: 5 },
              { value: "Fashion", count: 3 },
            ],
          },
          {
            field: "price_range",
            values: [{ value: "under_1000", count: 2 }],
          },
        ]}
        buildCategoryHref={(v) => `/search?category=${v}`}
        buildPriceHref={(v) => `/search?price=${v}`}
        clearHref="/search"
        labels={labels}
        priceRangeLabels={priceRangeLabels}
      />
    );
    expect(screen.getByText("Electronics")).toBeInTheDocument();
    expect(screen.getByText("(5)")).toBeInTheDocument();
    expect(screen.getByText("< ¥1,000")).toBeInTheDocument();
  });

  it("highlights the active facet and shows the clear link", () => {
    render(
      <SearchFiltersPanelPresenter
        facets={[
          {
            field: "category",
            values: [{ value: "Electronics", count: 5 }],
          },
        ]}
        activeCategoryId="Electronics"
        buildCategoryHref={(v) => `/search?category=${v}`}
        buildPriceHref={(v) => `/search?price=${v}`}
        clearHref="/search"
        labels={labels}
        priceRangeLabels={priceRangeLabels}
      />
    );
    expect(screen.getByRole("link", { name: "条件をクリア" })).toHaveAttribute("href", "/search");
    expect(screen.getByText("Electronics").closest("a")).toHaveClass("text-blue-600");
  });
});
