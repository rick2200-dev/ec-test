import type { Metadata } from "next";
import "./globals.css";
import AdminSidebar from "@/components/AdminSidebar";

export const metadata: Metadata = {
  title: "管理コンソール - EC Marketplace",
  description: "EC Marketplace プラットフォーム管理コンソール",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ja">
      <body className="bg-surface text-text-primary">
        <div className="flex min-h-screen">
          <AdminSidebar />
          <div className="flex-1 flex flex-col">
            {/* Top bar */}
            <header className="h-16 bg-white border-b border-border flex items-center justify-between px-6 shadow-sm">
              <h1 className="text-lg font-semibold text-text-primary">
                EC Marketplace 管理コンソール
              </h1>
              <div className="flex items-center gap-4">
                <button className="relative p-2 rounded-lg hover:bg-surface-hover transition-colors">
                  <svg
                    className="w-5 h-5 text-text-secondary"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
                    />
                  </svg>
                  <span className="absolute top-1 right-1 w-2 h-2 bg-danger rounded-full"></span>
                </button>
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center text-white text-sm font-medium">
                    A
                  </div>
                  <span className="text-sm font-medium text-text-primary">
                    管理者
                  </span>
                </div>
              </div>
            </header>
            {/* Main content */}
            <main className="flex-1 p-6">{children}</main>
          </div>
        </div>
      </body>
    </html>
  );
}
