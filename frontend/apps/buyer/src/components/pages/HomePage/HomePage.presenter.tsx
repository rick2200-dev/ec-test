import type { ReactNode } from "react";
import Link from "next/link";
import { ProductCardPresenter, type ProductCardPresenterProps } from "@/components/ProductCard";

export interface HomePageCategoryItem {
  id: string;
  href: string;
  name: string;
  /** SVG path d="..." for the category icon */
  iconPath: string;
  viewProductsLabel: string;
}

export interface HomePageProductItem extends ProductCardPresenterProps {
  id: string;
}

export interface HomePagePresenterProps {
  hero: {
    titleLines: string[];
    subtitle: string;
    ctaHref: string;
    ctaLabel: string;
  };
  categoriesSection: {
    title: string;
    items: HomePageCategoryItem[];
  };
  popularSection: {
    title: string;
    showAllHref: string;
    showAllLabel: string;
    products: HomePageProductItem[];
  };
  /** Slot for the (interactive) recommendations component */
  recommendationsSlot?: ReactNode;
}

export function HomePagePresenter({
  hero,
  categoriesSection,
  popularSection,
  recommendationsSlot,
}: HomePagePresenterProps) {
  return (
    <div>
      {/* Hero section */}
      <section className="bg-gradient-to-br from-blue-600 to-blue-800 text-white">
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6 lg:px-8">
          <div className="text-center">
            <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
              {hero.titleLines.map((line, i) => (
                <span key={i}>
                  {line}
                  {i < hero.titleLines.length - 1 && <br />}
                </span>
              ))}
            </h1>
            <p className="mt-4 text-lg text-blue-100">{hero.subtitle}</p>
            <div className="mt-8">
              <Link
                href={hero.ctaHref}
                className="inline-block rounded-full bg-white px-8 py-3 text-sm font-semibold text-blue-600 shadow-sm hover:bg-blue-50 transition-colors"
              >
                {hero.ctaLabel}
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Featured categories */}
      <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold text-gray-900">{categoriesSection.title}</h2>
        <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
          {categoriesSection.items.map((category) => (
            <Link
              key={category.id}
              href={category.href}
              className="group flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-6 transition-shadow hover:shadow-md"
            >
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 group-hover:bg-blue-100 transition-colors">
                <svg
                  aria-hidden="true"
                  className="h-6 w-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d={category.iconPath}
                  />
                </svg>
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">{category.name}</h3>
                <p className="text-sm text-gray-500">{category.viewProductsLabel}</p>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Popular products */}
      <section className="mx-auto max-w-7xl px-4 pb-16 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-gray-900">{popularSection.title}</h2>
          <Link
            href={popularSection.showAllHref}
            className="text-sm font-medium text-blue-600 hover:text-blue-800"
          >
            {popularSection.showAllLabel} &rarr;
          </Link>
        </div>
        <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {popularSection.products.map(({ id, ...cardProps }) => (
            <ProductCardPresenter key={id} {...cardProps} />
          ))}
        </div>
      </section>

      {recommendationsSlot}
    </div>
  );
}
