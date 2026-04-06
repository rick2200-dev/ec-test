import Link from "next/link";
import ProductCard from "@/components/ProductCard";
import { products, categories } from "@/lib/mock-data";

const categoryIcons: Record<string, string> = {
  electronics: "M 9 3 v 2 m 6 -2 v 2 M 9 19 v 2 m 6 -2 v 2 M 5 9 H 3 m 2 6 H 3 m 18 -6 h -2 m 2 6 h -2 M 7 19 h 10 a 2 2 0 0 0 2 -2 V 7 a 2 2 0 0 0 -2 -2 H 7 a 2 2 0 0 0 -2 2 v 10 a 2 2 0 0 0 2 2 Z m 4 -10 v 4 h 4",
  fashion: "M 15.75 10.5 l 4.72 -4.72 a 0.75 0.75 0 0 0 -1.28 -0.53 l -4.72 4.72 M 12 18.75 a 6 6 0 0 0 6 -6 v -1.5 m -6 7.5 a 6 6 0 0 1 -6 -6 v -1.5 m 6 7.5 v 3.75 m -3.75 0 h 7.5",
  handmade: "M 9.53 16.122 a 3 3 0 0 0 -5.78 1.128 2.25 2.25 0 0 1 -2.4 2.245 4.5 4.5 0 0 0 8.4 -2.245 c 0 -.399 -.078 -.78 -.22 -1.128 Z m 0 0 a 15.998 15.998 0 0 0 3.388 -1.62 m -5.043 -.025 a 15.994 15.994 0 0 1 1.622 -3.395 m 3.42 3.42 a 15.995 15.995 0 0 0 4.764 -4.648 l 3.876 -5.814 a 1.151 1.151 0 0 0 -1.597 -1.597 L 14.146 6.32 a 15.996 15.996 0 0 0 -4.649 4.763 m 3.42 3.42 a 6.776 6.776 0 0 0 -3.42 -3.42",
};

export default function HomePage() {
  const featuredProducts = products.filter((p) => p.status === "active").slice(0, 6);

  return (
    <div>
      {/* Hero section */}
      <section className="bg-gradient-to-br from-blue-600 to-blue-800 text-white">
        <div className="mx-auto max-w-7xl px-4 py-20 sm:px-6 lg:px-8">
          <div className="text-center">
            <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">
              あなたの欲しいものが
              <br />
              ここにある
            </h1>
            <p className="mt-4 text-lg text-blue-100">
              厳選されたセラーが届ける、高品質な商品をお楽しみください
            </p>
            <div className="mt-8">
              <Link
                href="/products"
                className="inline-block rounded-full bg-white px-8 py-3 text-sm font-semibold text-blue-600 shadow-sm hover:bg-blue-50 transition-colors"
              >
                商品を探す
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Featured categories */}
      <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold text-gray-900">カテゴリ</h2>
        <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
          {categories.map((category) => (
            <Link
              key={category.id}
              href={`/products?category=${category.slug}`}
              className="group flex items-center gap-4 rounded-xl border border-gray-200 bg-white p-6 transition-shadow hover:shadow-md"
            >
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 group-hover:bg-blue-100 transition-colors">
                <svg
                  className="h-6 w-6"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={1.5}
                    d={categoryIcons[category.slug] ?? "M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"}
                  />
                </svg>
              </div>
              <div>
                <h3 className="font-semibold text-gray-900">{category.name}</h3>
                <p className="text-sm text-gray-500">商品を見る</p>
              </div>
            </Link>
          ))}
        </div>
      </section>

      {/* Popular products */}
      <section className="mx-auto max-w-7xl px-4 pb-16 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between">
          <h2 className="text-2xl font-bold text-gray-900">人気の商品</h2>
          <Link
            href="/products"
            className="text-sm font-medium text-blue-600 hover:text-blue-800"
          >
            すべて見る &rarr;
          </Link>
        </div>
        <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {featuredProducts.map((product) => (
            <ProductCard key={product.id} product={product} />
          ))}
        </div>
      </section>
    </div>
  );
}
