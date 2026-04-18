import type { Meta, StoryObj } from "@storybook/react";
import { SearchResultsGridPresenter } from "./SearchResultsGrid.presenter";
import { SearchFiltersPanelPresenter } from "./SearchFiltersPanel.presenter";

const meta: Meta<typeof SearchResultsGridPresenter> = {
  component: SearchResultsGridPresenter,
  title: "Pages/SearchPage/ResultsGrid",
};
export default meta;

type Story = StoryObj<typeof SearchResultsGridPresenter>;

export const Empty: Story = {
  args: { results: [], emptyLabel: "検索結果が見つかりませんでした" },
};

export const WithResults: Story = {
  args: {
    emptyLabel: "",
    results: [
      {
        id: "p1",
        seller_id: "s1",
        name: "Premium Wireless Headphones",
        slug: "premium-wireless-headphones",
        description: "",
        status: "active",
        price_amount: 15800,
        price_currency: "JPY",
        seller_name: "Tokyo Electronics",
        category_name: "Electronics",
        score: 2.5,
        is_promoted: false,
        plan_tier: 2,
      },
      {
        id: "p2",
        seller_id: "s2",
        name: "Organic Cotton T-Shirt",
        slug: "organic-cotton-tshirt",
        description: "",
        status: "active",
        price_amount: 3200,
        price_currency: "JPY",
        seller_name: "Osaka Fashion",
        category_name: "Fashion",
        score: 1.5,
        is_promoted: false,
        plan_tier: 1,
      },
    ],
  },
};

export const Filters: StoryObj<typeof SearchFiltersPanelPresenter> = {
  render: (args) => <SearchFiltersPanelPresenter {...args} />,
  args: {
    facets: [
      {
        field: "category",
        values: [
          { value: "Electronics", count: 24 },
          { value: "Fashion", count: 18 },
        ],
      },
      {
        field: "price_range",
        values: [
          { value: "under_1000", count: 4 },
          { value: "1000_5000", count: 12 },
        ],
      },
    ],
    buildCategoryHref: (v) => `/search?category=${v}`,
    buildPriceHref: (v) => `/search?price=${v}`,
    clearHref: "/search",
    labels: {
      categories: "カテゴリ",
      priceRange: "価格帯",
      clearFilters: "条件をクリア",
    },
    priceRangeLabels: {
      under_1000: "< ¥1,000",
      "1000_5000": "¥1,000 - ¥5,000",
    },
  },
};
