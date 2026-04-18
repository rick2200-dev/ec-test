import type { ReactNode } from "react";

export type PointsPageTab = "transactions" | "coupons";

export interface PointsPagePresenterProps {
  title: string;
  tabs: { id: PointsPageTab; label: string }[];
  activeTab: PointsPageTab;
  onTabChange: (tab: PointsPageTab) => void;
  balanceSlot: ReactNode;
  contentSlot: ReactNode;
  errorSlot?: ReactNode;
}

export function PointsPagePresenter({
  title,
  tabs,
  activeTab,
  onTabChange,
  balanceSlot,
  contentSlot,
  errorSlot,
}: PointsPagePresenterProps) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-2xl font-bold text-gray-900">{title}</h1>

      {errorSlot}

      <div className="mt-6">{balanceSlot}</div>

      <div className="mt-8 border-b border-gray-200">
        <nav className="-mb-px flex gap-6" aria-label="Tabs">
          {tabs.map((tab) => {
            const active = tab.id === activeTab;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => onTabChange(tab.id)}
                className={`whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium transition-colors ${
                  active
                    ? "border-blue-600 text-blue-600"
                    : "border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700"
                }`}
                aria-current={active ? "page" : undefined}
              >
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      <div className="mt-6">{contentSlot}</div>
    </div>
  );
}
