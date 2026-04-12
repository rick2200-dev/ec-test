import Link from "next/link";

export interface NotificationCardItem {
  id: string;
  /** Pre-formatted localized timestamp label (e.g. "2026年4月10日 14:30"). */
  createdAtLabel: string;
  /** Localized headline (e.g. "商品が発送されました"). */
  title: string;
  /** Localized body copy already filled with variables. */
  body: string;
  /** Optional deep-link (e.g. `/orders/ord-1`). When omitted, the card is not clickable. */
  href?: string;
  /** Whether this notification is still unread — controls the dot + background emphasis. */
  unread: boolean;
}

export interface NotificationsPagePresenterProps {
  title: string;
  description: string;
  emptyMessage: string;
  unreadBadgeLabel: string;
  notifications: NotificationCardItem[];
}

/**
 * NotificationsPagePresenter renders the buyer `/notifications` index route.
 * Pure presentation component — the container is responsible for resolving
 * translations, formatting dates, and assembling the card list.
 *
 * Empty state: shows a friendly "no notifications" message without a CTA,
 * since there is no user action that populates notifications directly.
 */
export function NotificationsPagePresenter({
  title,
  description,
  emptyMessage,
  unreadBadgeLabel,
  notifications,
}: NotificationsPagePresenterProps) {
  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
        <p className="mt-1 text-sm text-gray-500">{description}</p>
      </div>

      {notifications.length === 0 ? (
        <div className="mt-8 flex flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center">
          <svg
            aria-hidden="true"
            className="h-12 w-12 text-gray-300"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M14.857 17.082a23.848 23.848 0 005.454-1.31A8.967 8.967 0 0118 9.75v-.7V9A6 6 0 006 9v.75a8.967 8.967 0 01-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 01-5.714 0m5.714 0a3 3 0 11-5.714 0"
            />
          </svg>
          <p className="mt-4 text-sm text-gray-500">{emptyMessage}</p>
        </div>
      ) : (
        <ul className="mt-6 space-y-3">
          {notifications.map((n) => {
            const card = (
              <div
                className={`rounded-lg border p-4 transition-shadow ${
                  n.unread ? "border-blue-200 bg-blue-50/40" : "border-gray-200 bg-white"
                } ${n.href ? "hover:shadow-md" : ""}`}
              >
                <div className="flex items-start gap-3">
                  {/* Unread dot */}
                  <span
                    aria-hidden="true"
                    className={`mt-1.5 inline-block h-2 w-2 flex-shrink-0 rounded-full ${
                      n.unread ? "bg-blue-600" : "bg-transparent"
                    }`}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <p className="text-sm font-semibold text-gray-900">{n.title}</p>
                      {n.unread && (
                        <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-[10px] font-medium text-blue-700">
                          {unreadBadgeLabel}
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-sm text-gray-600">{n.body}</p>
                    <p className="mt-2 text-xs text-gray-400">{n.createdAtLabel}</p>
                  </div>
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
