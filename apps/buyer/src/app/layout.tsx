import type { Metadata } from "next";
import Link from "next/link";
import { getLocale, getTranslations } from "next-intl/server";
import { NextIntlClientProvider } from "next-intl";
import "./globals.css";

export const metadata: Metadata = {
  title: "EC Marketplace",
  description: "あなたの欲しいものが見つかるマーケットプレイス",
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const locale = await getLocale();
  const t = await getTranslations();

  return (
    <html lang={locale}>
      <body className="min-h-screen flex flex-col">
        {/* Skip to content */}
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:absolute focus:z-[100] focus:bg-white focus:px-4 focus:py-2 focus:text-blue-600 focus:underline"
        >
          {t("a11y.skipToContent")}
        </a>

        <NextIntlClientProvider>
          {/* Header */}
          <header className="sticky top-0 z-50 bg-white border-b border-gray-200">
            <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
              <div className="flex h-16 items-center justify-between gap-4">
                {/* Logo */}
                <Link href="/" className="shrink-0">
                  <span className="text-xl font-bold text-blue-600">EC Marketplace</span>
                </Link>

                {/* Search bar */}
                <search className="flex-1 max-w-xl">
                  <div className="relative">
                    <input
                      type="text"
                      placeholder={t("header.searchPlaceholder")}
                      className="w-full rounded-full border border-gray-300 bg-gray-50 py-2 pl-4 pr-10 text-sm focus:border-blue-500 focus:bg-white focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                    <button
                      aria-label={t("a11y.searchProducts")}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                    >
                      <svg
                        aria-hidden="true"
                        className="h-5 w-5"
                        fill="none"
                        stroke="currentColor"
                        viewBox="0 0 24 24"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
                        />
                      </svg>
                    </button>
                  </div>
                </search>

                {/* Navigation */}
                <nav className="flex items-center gap-4">
                  <Link href="/products" className="text-sm text-gray-600 hover:text-gray-900">
                    {t("nav.products")}
                  </Link>
                  {/* Cart icon */}
                  <button
                    aria-label={t("a11y.cart")}
                    className="relative text-gray-600 hover:text-gray-900"
                  >
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
                        strokeWidth={2}
                        d="M2.25 3h1.386c.51 0 .955.343 1.087.835l.383 1.437M7.5 14.25a3 3 0 0 0-3 3h15.75m-12.75-3h11.218c1.121-2.3 2.1-4.684 2.924-7.138a60.114 60.114 0 0 0-16.536-1.84M7.5 14.25 5.106 5.272M6 20.25a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Zm12.75 0a.75.75 0 1 1-1.5 0 .75.75 0 0 1 1.5 0Z"
                      />
                    </svg>
                    <span className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-blue-600 text-[10px] font-bold text-white">
                      0
                    </span>
                  </button>
                </nav>
              </div>
            </div>
          </header>

          {/* Main content */}
          <main id="main-content" className="flex-1">
            {children}
          </main>

          {/* Footer */}
          <footer className="border-t border-gray-200 bg-gray-50">
            <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
              <div className="grid grid-cols-1 gap-8 sm:grid-cols-3">
                <div>
                  <h3 className="text-sm font-semibold text-gray-900">EC Marketplace</h3>
                  <p className="mt-2 text-sm text-gray-500">{t("footer.description")}</p>
                </div>
                <nav aria-label={t("footer.categories")}>
                  <h3 className="text-sm font-semibold text-gray-900">{t("footer.categories")}</h3>
                  <ul className="mt-2 space-y-1">
                    <li>
                      <Link
                        href="/products?category=electronics"
                        className="text-sm text-gray-500 hover:text-gray-700"
                      >
                        Electronics
                      </Link>
                    </li>
                    <li>
                      <Link
                        href="/products?category=fashion"
                        className="text-sm text-gray-500 hover:text-gray-700"
                      >
                        Fashion
                      </Link>
                    </li>
                    <li>
                      <Link
                        href="/products?category=handmade"
                        className="text-sm text-gray-500 hover:text-gray-700"
                      >
                        Handmade
                      </Link>
                    </li>
                  </ul>
                </nav>
                <nav aria-label={t("footer.support")}>
                  <h3 className="text-sm font-semibold text-gray-900">{t("footer.support")}</h3>
                  <ul className="mt-2 space-y-1">
                    <li>
                      <span className="text-sm text-gray-500">{t("footer.helpCenter")}</span>
                    </li>
                    <li>
                      <span className="text-sm text-gray-500">{t("footer.contact")}</span>
                    </li>
                    <li>
                      <span className="text-sm text-gray-500">{t("footer.terms")}</span>
                    </li>
                  </ul>
                </nav>
              </div>
              <div className="mt-8 border-t border-gray-200 pt-4">
                <p className="text-center text-xs text-gray-400">{t("footer.copyright")}</p>
              </div>
            </div>
          </footer>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
