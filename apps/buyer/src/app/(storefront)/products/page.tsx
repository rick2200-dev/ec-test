import ProductCard from "@/components/ProductCard";
import Link from "next/link";
import {
  products,
  categories,
  getCategoryById,
} from "@/lib/mock-data";

interface ProductsPageProps {
  searchParams: Promise<{ category?: string; sort?: string; page?: string }>;
}

export default async function ProductsPage({ searchParams }: ProductsPageProps) {
  const params = await searchParams;
  const categorySlug = params.category;
  const currentPage = Number(params.page) || 1;
  const perPage = 9;

  // Filter products
  let filtered = products.filter((p) => p.status === "active");

  if (categorySlug) {
    const cat = categories.find((c) => c.slug === categorySlug);
    if (cat) {
      filtered = filtered.filter((p) => p.category_id === cat.id);
    }
  }

  const totalPages = Math.max(1, Math.ceil(filtered.length / perPage));
  const paged = filtered.slice((currentPage - 1) * perPage, currentPage * perPage);

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-2xl font-bold text-gray-900">商品一覧</h1>

      <div className="mt-6 flex flex-col gap-8 lg:flex-row">
        {/* Sidebar - Category filters */}
        <aside className="w-full shrink-0 lg:w-56">
          <div className="rounded-lg border border-gray-200 bg-white p-4">
            <h2 className="font-semibold text-gray-900">カテゴリ</h2>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/products"
                  className={`block text-sm ${
                    !categorySlug
                      ? "font-semibold text-blue-600"
                      : "text-gray-600 hover:text-gray-900"
                  }`}
                >
                  すべて
                </Link>
              </li>
              {categories.map((cat) => (
                <li key={cat.id}>
                  <Link
                    href={`/products?category=${cat.slug}`}
                    className={`block text-sm ${
                      categorySlug === cat.slug
                        ? "font-semibold text-blue-600"
                        : "text-gray-600 hover:text-gray-900"
                    }`}
                  >
                    {cat.name}
                  </Link>
                </li>
              ))}
            </ul>
          </div>

          {/* Sort placeholder */}
          <div className="mt-4 rounded-lg border border-gray-200 bg-white p-4">
            <h2 className="font-semibold text-gray-900">並び替え</h2>
            <select className="mt-2 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500">
              <option>おすすめ順</option>
              <option>価格が安い順</option>
              <option>価格が高い順</option>
              <option>新着順</option>
            </select>
          </div>
        </aside>

        {/* Product grid */}
        <div className="flex-1">
          {paged.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-gray-500">
              <svg
                className="h-12 w-12 text-gray-300"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
                />
              </svg>
              <p className="mt-4 text-sm">該当する商品が見つかりませんでした</p>
            </div>
          ) : (
            <>
              <p className="mb-4 text-sm text-gray-500">
                {filtered.length}件の商品
              </p>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
                {paged.map((product) => (
                  <ProductCard key={product.id} product={product} />
                ))}
              </div>
            </>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-8 flex items-center justify-center gap-2">
              {Array.from({ length: totalPages }, (_, i) => i + 1).map(
                (page) => (
                  <Link
                    key={page}
                    href={`/products?${categorySlug ? `category=${categorySlug}&` : ""}page=${page}`}
                    className={`inline-flex h-9 w-9 items-center justify-center rounded-md text-sm ${
                      page === currentPage
                        ? "bg-blue-600 text-white"
                        : "border border-gray-300 text-gray-600 hover:bg-gray-50"
                    }`}
                  >
                    {page}
                  </Link>
                )
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
