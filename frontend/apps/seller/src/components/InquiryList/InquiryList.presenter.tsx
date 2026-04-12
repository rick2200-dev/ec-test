import Link from "next/link";

export interface InquiryListItem {
  id: string;
  href: string;
  productName: string;
  skuCode: string;
  subject: string;
  lastMessageAt: string;
  status: "open" | "closed";
  statusLabel: string;
  unreadCount: number;
}

export interface InquiryListPresenterProps {
  title: string;
  description: string;
  emptyLabel: string;
  productColumnLabel: string;
  lastMessageColumnLabel: string;
  statusColumnLabel: string;
  unreadLabel: string;
  items: InquiryListItem[];
}

/**
 * Seller-app inquiry list presenter. Mirrors the buyer app variant with
 * seller-specific accent colors. Kept separate per plan until a shared
 * UI package is justified.
 */
export function InquiryListPresenter({
  title,
  description,
  emptyLabel,
  productColumnLabel,
  lastMessageColumnLabel,
  statusColumnLabel,
  unreadLabel,
  items,
}: InquiryListPresenterProps) {
  return (
    <section className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{title}</h2>
        <p className="mt-1 text-text-secondary">{description}</p>
      </div>

      {items.length === 0 ? (
        <div
          className="rounded-lg border border-dashed border-border bg-white py-12 text-center text-sm text-text-secondary"
          role="status"
        >
          {emptyLabel}
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-border shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-surface">
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {productColumnLabel}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {lastMessageColumnLabel}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {statusColumnLabel}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {items.map((item) => (
                  <tr key={item.id} className="hover:bg-surface-hover transition-colors">
                    <td className="px-6 py-4">
                      <Link href={item.href} className="block">
                        <div className="text-sm font-medium text-accent hover:text-accent-dark">
                          {item.subject}
                        </div>
                        <div className="mt-0.5 text-xs text-text-secondary">
                          {item.productName}
                          <span className="ml-2 font-mono text-text-muted">{item.skuCode}</span>
                        </div>
                      </Link>
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{item.lastMessageAt}</td>
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            item.status === "open"
                              ? "bg-green-100 text-green-800"
                              : "bg-gray-100 text-gray-600"
                          }`}
                        >
                          {item.statusLabel}
                        </span>
                        {item.unreadCount > 0 && (
                          <span
                            className="inline-flex items-center rounded-full bg-accent px-2 py-0.5 text-xs font-semibold text-white"
                            aria-label={unreadLabel}
                          >
                            {item.unreadCount}
                          </span>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  );
}
