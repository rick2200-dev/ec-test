import Link from "next/link";

export interface PromotedBannerItem {
  id: string;
  href: string;
  name: string;
  sellerName: string;
  priceLabel: string;
}

export interface PromotedBannerPresenterProps {
  items: PromotedBannerItem[];
  sponsoredLabel: string;
}

export function PromotedBannerPresenter({ items, sponsoredLabel }: PromotedBannerPresenterProps) {
  if (items.length === 0) return null;

  return (
    <div className="mb-8">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {items.map((item) => (
          <Link
            key={item.id}
            href={item.href}
            className="group block rounded-lg border border-amber-200 bg-amber-50 overflow-hidden transition-shadow hover:shadow-md"
          >
            <div className="aspect-[3/2] bg-amber-100 flex items-center justify-center relative">
              <svg
                aria-hidden="true"
                className="w-12 h-12 text-amber-300"
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
              <span className="absolute top-2 left-2 px-2 py-0.5 bg-amber-500 text-white text-xs font-medium rounded">
                {sponsoredLabel}
              </span>
            </div>
            <div className="p-3">
              <h4 className="text-sm font-medium text-gray-900 group-hover:text-blue-600 transition-colors line-clamp-1">
                {item.name}
              </h4>
              <p className="text-xs text-gray-500 mt-0.5">{item.sellerName}</p>
              <p className="mt-1 text-base font-bold text-gray-900">{item.priceLabel}</p>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
