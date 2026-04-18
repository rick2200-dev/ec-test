import type { Metadata } from "next";
import Link from "next/link";
import { getLocale, getTranslations } from "next-intl/server";
import { NextIntlClientProvider } from "next-intl";
import { auth0 } from "@/lib/auth0";
import CartBadge from "@/components/CartBadge";
import SearchBar from "@/components/SearchBar";
import "./globals.css";

export const metadata: Metadata = {
  title: "EC Marketplace",
  description: "あなたの欲しいものが見つかるマーケットプレイス",
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const locale = await getLocale();
  const t = await getTranslations();
  // Read the session server-side so login state SSRs without a flash of
  // "Log in" on first paint. The SDK middleware refreshes this cookie.
  // Wrap in try/catch because the SDK throws when Auth0 env is missing —
  // in that case we fall back to logged-out UI rather than crash the page.
  let user: { email?: string | null; name?: string | null } | null = null;
  try {
    const session = await auth0.getSession();
    user = session?.user ?? null;
  } catch {
    user = null;
  }

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

                {/* Search bar — now driven by /api/v1/buyer/search/suggest */}
                <div className="flex-1 max-w-xl">
                  <SearchBar />
                </div>

                {/* Navigation */}
                <nav className="flex items-center gap-4">
                  <Link href="/products" className="text-sm text-gray-600 hover:text-gray-900">
                    {t("nav.products")}
                  </Link>
                  {user ? (
                    <>
                      <span
                        className="max-w-[160px] truncate text-sm text-gray-700"
                        title={user.email ?? undefined}
                      >
                        {user.email ?? user.name}
                      </span>
                      <a href="/auth/logout" className="text-sm text-gray-600 hover:text-gray-900">
                        {t("auth.logout")}
                      </a>
                    </>
                  ) : (
                    <a href="/auth/login" className="text-sm text-gray-600 hover:text-gray-900">
                      {t("auth.login")}
                    </a>
                  )}
                  {/* Notification bell */}
                  <Link
                    href="/notifications"
                    aria-label={t("a11y.notifications")}
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
                        d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0"
                      />
                    </svg>
                    <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-red-500" />
                  </Link>
                  {/* Cart icon */}
                  <CartBadge />
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
