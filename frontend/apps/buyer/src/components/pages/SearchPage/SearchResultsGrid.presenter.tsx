import Link from "next/link";
import { formatCurrency } from "@ec-marketplace/ui-utils";
import type { ProductSearchHit } from "@ec-marketplace/types";

export interface SearchResultsGridPresenterProps {
  results: ProductSearchHit[];
  emptyLabel: string;
}

export function SearchResultsGridPresenter({
  results,
  emptyLabel,
}: SearchResultsGridPresenterProps) {
  if (results.length === 0) {
    return <p className="py-12 text-center text-gray-500">{emptyLabel}</p>;
  }
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {results.map((p) => (
        <Link
          key={p.id}
          href={`/products/${p.slug}`}
          className="group block overflow-hidden rounded-lg border border-gray-200 bg-white transition-shadow hover:shadow-md"
        >
          <div className="flex aspect-square items-center justify-center bg-gray-100">
            <svg
              aria-hidden="true"
              className="h-16 w-16 text-gray-300"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0 0 22.5 18.75V5.25A2.25 2.25 0 0 0 20.25 3H3.75A2.25 2.25 0 0 0 1.5 5.25v13.5A2.25 2.25 0 0 0 3.75 21Z"
              />
            </svg>
          </div>
          <div className="p-4">
            <h3 className="line-clamp-2 text-sm font-medium text-gray-900 transition-colors group-hover:text-blue-600">
              {p.name}
            </h3>
            <p className="mt-1 text-xs text-gray-500">{p.seller_name}</p>
            <p className="mt-2 text-lg font-bold text-gray-900">
              {p.price_currency === "JPY"
                ? formatCurrency(Math.round(p.price_amount))
                : `${p.price_amount.toLocaleString()} ${p.price_currency}`}
            </p>
          </div>
        </Link>
      ))}
    </div>
  );
}
