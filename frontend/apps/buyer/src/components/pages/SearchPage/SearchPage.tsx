"use client";

import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import PromotedBanner from "@/components/PromotedBanner";
import { searchProducts } from "@/lib/api";
import type { SearchResponse } from "@ec-marketplace/types";
import { SearchFiltersPanelPresenter } from "./SearchFiltersPanel.presenter";
import { SearchResultsGridPresenter } from "./SearchResultsGrid.presenter";

const PRICE_RANGES: Record<string, { min?: number; max?: number }> = {
  under_1000: { max: 1000 },
  "1000_5000": { min: 1000, max: 5000 },
  "5000_10000": { min: 5000, max: 10000 },
  "10000_50000": { min: 10000, max: 50000 },
  "50000_plus": { min: 50000 },
};

const PAGE_SIZE = 24;

function buildHref(qs: URLSearchParams, overrides: Record<string, string | undefined>): string {
  const next = new URLSearchParams(qs);
  for (const [k, v] of Object.entries(overrides)) {
    if (v == null || v === "") next.delete(k);
    else next.set(k, v);
  }
  next.delete("offset");
  return `/search?${next.toString()}`;
}

export default function SearchPage() {
  const t = useTranslations();
  const searchParams = useSearchParams();
  const query = searchParams.get("q") ?? "";
  const category = searchParams.get("category") ?? "";
  const priceRange = searchParams.get("price") ?? "";
  const sortBy = searchParams.get("sort") ?? "";
  const offset = parseInt(searchParams.get("offset") ?? "0", 10) || 0;

  const [data, setData] = useState<SearchResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const range = PRICE_RANGES[priceRange];
    (async () => {
      try {
        const res = await searchProducts({
          q: query,
          category_id: category || undefined,
          min_price: range?.min,
          max_price: range?.max,
          sort_by: sortBy || undefined,
          limit: PAGE_SIZE,
          offset,
        });
        if (!cancelled) setData(res);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("search.errorGeneric"));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [query, category, priceRange, sortBy, offset, t]);

  const total = data?.total ?? 0;
  const results = data?.products ?? [];
  const promoted = data?.promoted_products ?? [];
  const facets = data?.facets ?? [];

  const baseQS = useMemo(() => new URLSearchParams(searchParams.toString()), [searchParams]);
  const buildCategoryHref = (value: string) =>
    buildHref(baseQS, { category: value === category ? undefined : value });
  const buildPriceHref = (value: string) =>
    buildHref(baseQS, { price: value === priceRange ? undefined : value });
  const clearHref = buildHref(baseQS, { category: undefined, price: undefined });

  const priceRangeLabels: Record<string, string> = {
    under_1000: "< ¥1,000",
    "1000_5000": "¥1,000 - ¥5,000",
    "5000_10000": "¥5,000 - ¥10,000",
    "10000_50000": "¥10,000 - ¥50,000",
    "50000_plus": "¥50,000+",
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">
          {query ? t("search.resultsFor", { query }) : t("search.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          {loading ? t("search.loading") : t("search.totalResults", { count: total })}
        </p>
      </div>

      {error && (
        <div
          role="alert"
          className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
        >
          {error}
        </div>
      )}

      <div className="flex gap-8">
        <SearchFiltersPanelPresenter
          facets={facets}
          activeCategoryId={category || undefined}
          activePriceRange={priceRange || undefined}
          buildCategoryHref={buildCategoryHref}
          buildPriceHref={buildPriceHref}
          clearHref={clearHref}
          labels={{
            categories: t("search.categories"),
            priceRange: t("search.priceRange"),
            clearFilters: t("search.clearFilters"),
          }}
          priceRangeLabels={priceRangeLabels}
        />

        <div className="flex-1">
          <PromotedBanner products={promoted} />
          <SearchResultsGridPresenter results={results} emptyLabel={t("search.noResults")} />

          {total > PAGE_SIZE && (
            <div className="mt-6 flex items-center justify-between text-sm">
              {offset > 0 && (
                <a
                  href={buildHref(baseQS, { offset: String(Math.max(0, offset - PAGE_SIZE)) })}
                  className="rounded-md border border-gray-300 px-3 py-2 hover:bg-gray-50"
                >
                  {t("search.prevPage")}
                </a>
              )}
              <span className="text-gray-500">
                {offset + 1} - {Math.min(offset + PAGE_SIZE, total)} / {total}
              </span>
              {offset + PAGE_SIZE < total && (
                <a
                  href={buildHref(baseQS, { offset: String(offset + PAGE_SIZE) })}
                  className="rounded-md border border-gray-300 px-3 py-2 hover:bg-gray-50"
                >
                  {t("search.nextPage")}
                </a>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
