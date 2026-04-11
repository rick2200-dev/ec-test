import Link from "next/link";

export type NotificationCategory =
  | "order"
  | "inventory"
  | "inquiry"
  | "account";

export interface NotificationCardItem {
  id: string;
  /** Pre-formatted timestamp label (e.g. "2026/04/11 18:12"). */
  createdAtLabel: string;
  title: string;
  body: string;
  category: NotificationCategory;
  unread: boolean;
  href?: string;
}

export type NotificationFilter = "all" | "unread";

export interface NotificationsPagePresenterProps {
  title: string;
  description: string;
  emptyMessage: string;
  unreadBadgeLabel: string;
  markAllReadLabel: string;
  filterAllLabel: string;
  filterUnreadLabel: string;
  filter: NotificationFilter;
  onFilterChange: (filter: NotificationFilter) => void;
  onMarkAllRead: () => void;
  notifications: NotificationCardItem[];
}

const CATEGORY_ICONS: Record<NotificationCategory, string> = {
  // Path shapes chosen to match the seller app's icon vocabulary (stroke-only SVGs).
  order:
    "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z",
  inventory:
    "M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4",
  inquiry:
    "M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z",
  account:
    "M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z",
};

/**
 * SellerNotificationsPagePresenter renders the seller `/notifications` screen.
 * Styled to match the rest of the seller console (surface / border / accent tokens
 * from globals.css). Presentation only — filter and mark-all state are owned
 * by the container.
 */
export function NotificationsPagePresenter({
  title,
  description,
  emptyMessage,
  unreadBadgeLabel,
  markAllReadLabel,
  filterAllLabel,
  filterUnreadLabel,
  filter,
  onFilterChange,
  onMarkAllRead,
  notifications,
}: NotificationsPagePresenterProps) {
  const hasUnread = notifications.some((n) => n.unread);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">{title}</h2>
          <p className="text-text-secondary mt-1">{description}</p>
        </div>
        <button
          type="button"
          onClick={onMarkAllRead}
          disabled={!hasUnread}
          className="rounded-md border border-border bg-white px-3 py-1.5 text-sm font-medium text-text-primary transition-colors hover:bg-surface-hover disabled:cursor-not-allowed disabled:opacity-50"
        >
          {markAllReadLabel}
        </button>
      </div>

      {/* Filter tabs */}
      <div className="flex items-center gap-1 border-b border-border" role="tablist">
        {(
          [
            { key: "all" as const, label: filterAllLabel },
            { key: "unread" as const, label: filterUnreadLabel },
          ]
        ).map((tab) => (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={filter === tab.key}
            onClick={() => onFilterChange(tab.key)}
            className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
              filter === tab.key
                ? "border-accent text-accent"
                : "border-transparent text-text-secondary hover:text-text-primary"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* List */}
      {notifications.length === 0 ? (
        <div
          className="rounded-lg border border-dashed border-border bg-white px-6 py-16 text-center text-text-secondary"
          role="status"
        >
          {emptyMessage}
        </div>
      ) : (
        <ul className="space-y-3">
          {notifications.map((n) => {
            const card = (
              <div
                className={`flex items-start gap-4 rounded-lg border p-4 transition-shadow ${
                  n.unread
                    ? "border-accent/30 bg-accent/5"
                    : "border-border bg-white"
                } ${n.href ? "hover:shadow-sm" : ""}`}
              >
                <div
                  className={`flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full ${
                    n.unread ? "bg-accent/10 text-accent" : "bg-surface text-text-secondary"
                  }`}
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
                      d={CATEGORY_ICONS[n.category]}
                    />
                  </svg>
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="text-sm font-semibold text-text-primary">{n.title}</p>
                    {n.unread && (
                      <span className="inline-flex items-center rounded-full bg-accent/10 px-2 py-0.5 text-[10px] font-medium text-accent">
                        {unreadBadgeLabel}
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-sm text-text-secondary">{n.body}</p>
                  <p className="mt-2 text-xs text-text-secondary/70">{n.createdAtLabel}</p>
                </div>
              </div>
            );

            return (
              <li key={n.id}>
                {n.href ? (
                  <Link href={n.href} className="block">
                    {card}
                  </Link>
                ) : (
                  card
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
