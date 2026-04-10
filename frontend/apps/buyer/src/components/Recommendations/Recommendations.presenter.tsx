import { ProductCardPresenter, type ProductCardPresenterProps } from "../ProductCard";

export interface RecommendationsItem extends ProductCardPresenterProps {
  id: string;
}

export interface RecommendationsPresenterProps {
  title: string;
  loading: boolean;
  items: RecommendationsItem[];
}

export function RecommendationsPresenter({
  title,
  loading,
  items,
}: RecommendationsPresenterProps) {
  if (loading) {
    return (
      <section
        aria-live="polite"
        aria-busy={true}
        className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8"
      >
        <h2 className="text-2xl font-bold text-gray-900">{title}</h2>
        <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              aria-hidden="true"
              className="animate-pulse rounded-lg border border-gray-200 bg-white overflow-hidden"
            >
              <div className="aspect-square bg-gray-200" />
              <div className="p-4 space-y-2">
                <div className="h-4 bg-gray-200 rounded w-3/4" />
                <div className="h-4 bg-gray-200 rounded w-1/2" />
              </div>
            </div>
          ))}
        </div>
      </section>
    );
  }

  if (items.length === 0) return null;

  return (
    <section
      aria-live="polite"
      aria-busy={false}
      className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8"
    >
      <h2 className="text-2xl font-bold text-gray-900">{title}</h2>
      <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
        {items.map(({ id, ...cardProps }) => (
          <ProductCardPresenter key={id} {...cardProps} />
        ))}
      </div>
    </section>
  );
}
