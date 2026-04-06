import Link from "next/link";
import { Product } from "@/lib/types";
import { formatPrice, getLowestPrice, getSellerById } from "@/lib/mock-data";

interface ProductCardProps {
  product: Product;
}

export default function ProductCard({ product }: ProductCardProps) {
  const price = getLowestPrice(product.id);
  const seller = getSellerById(product.seller_id);

  return (
    <Link
      href={`/products/${product.slug}`}
      className="group block rounded-lg border border-gray-200 bg-white overflow-hidden transition-shadow hover:shadow-md"
    >
      {/* Image placeholder */}
      <div className="aspect-square bg-gray-100 flex items-center justify-center">
        <svg
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
          {product.name}
        </h3>
        {seller && (
          <p className="mt-1 text-xs text-gray-500">{seller.name}</p>
        )}
        <p className="mt-2 text-lg font-bold text-gray-900">
          {formatPrice(price)}
        </p>
      </div>
    </Link>
  );
}
