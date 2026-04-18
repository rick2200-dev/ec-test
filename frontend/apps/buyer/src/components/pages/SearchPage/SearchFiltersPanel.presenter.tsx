import type { SearchFacet } from "@ec-marketplace/types";

export interface SearchFiltersPanelPresenterProps {
  facets: SearchFacet[];
  activeCategoryId?: string;
  activePriceRange?: string;
  buildCategoryHref: (value: string) => string;
  buildPriceHref: (value: string) => string;
  clearHref: string;
  labels: {
    categories: string;
    priceRange: string;
    clearFilters: string;
  };
  priceRangeLabels: Record<string, string>;
}

export function SearchFiltersPanelPresenter({
  facets,
  activeCategoryId,
  activePriceRange,
  buildCategoryHref,
  buildPriceHref,
  clearHref,
  labels,
  priceRangeLabels,
}: SearchFiltersPanelPresenterProps) {
  const categoryFacet = facets.find((f) => f.field === "category");
  const priceFacet = facets.find((f) => f.field === "price_range");
  const hasFilters = activeCategoryId || activePriceRange;

  return (
    <aside className="hidden w-56 shrink-0 lg:block">
      {hasFilters && (
        <a
          href={clearHref}
          className="mb-4 inline-block text-xs font-medium text-blue-600 hover:text-blue-800"
        >
          {labels.clearFilters}
        </a>
      )}
      {categoryFacet && categoryFacet.values.length > 0 && (
        <div className="mb-6">
          <h3 className="mb-3 text-sm font-semibold text-gray-900">{labels.categories}</h3>
          <ul className="space-y-2">
            {categoryFacet.values.map((v) => {
              const active = v.value === activeCategoryId;
              return (
                <li key={v.value}>
                  <a
                    href={buildCategoryHref(v.value)}
                    className={`flex justify-between text-sm hover:text-blue-600 ${
                      active ? "font-semibold text-blue-600" : "text-gray-600"
                    }`}
                  >
                    <span>{v.value}</span>
                    <span className="text-gray-400">({v.count})</span>
                  </a>
                </li>
              );
            })}
          </ul>
        </div>
      )}
      {priceFacet && priceFacet.values.length > 0 && (
        <div className="mb-6">
          <h3 className="mb-3 text-sm font-semibold text-gray-900">{labels.priceRange}</h3>
          <ul className="space-y-2">
            {priceFacet.values.map((v) => {
              const active = v.value === activePriceRange;
              return (
                <li key={v.value}>
                  <a
                    href={buildPriceHref(v.value)}
                    className={`flex justify-between text-sm hover:text-blue-600 ${
                      active ? "font-semibold text-blue-600" : "text-gray-600"
                    }`}
                  >
                    <span>{priceRangeLabels[v.value] ?? v.value}</span>
                    <span className="text-gray-400">({v.count})</span>
                  </a>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </aside>
  );
}
