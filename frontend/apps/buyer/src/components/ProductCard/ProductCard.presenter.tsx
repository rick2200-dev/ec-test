import Link from "next/link";

export interface ProductCardPresenterProps {
  href: string;
  name: string;
  sellerName?: string;
  priceLabel: string;
}

export function ProductCardPresenter({
  href,
  name,
  sellerName,
  priceLabel,
}: ProductCardPresenterProps) {
  return (
    <Link
      href={href}
      className="group block rounded-lg border border-gray-200 bg-white overflow-hidden transition-shadow hover:shadow-md"
    >
      {/* Image placeholder */}
      <div className="aspect-square bg-gray-100 flex items-center justify-center">
        <svg
          aria-hidden="true"
          className="w-16 h-16 text-gray-300"
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

      {/* Product info */}
      <div className="p-4">
        <h3 className="text-sm font-medium text-gray-900 group-hover:text-blue-600 transition-colors line-clamp-2">
          {name}
        </h3>
        {sellerName && <p className="mt-1 text-xs text-gray-500">{sellerName}</p>}
        <p className="mt-2 text-lg font-bold text-gray-900">{priceLabel}</p>
      </div>
    </Link>
  );
}
