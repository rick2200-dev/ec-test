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
 * Pure presenter for an inquiry thread list. Works for both the buyer
 * and seller apps — the container decides the href shape and labels.
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
    <section className="space-y-4">
      <header>
        <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
        <p className="mt-1 text-sm text-gray-500">{description}</p>
      </header>

      {items.length === 0 ? (
        <div
          className="rounded-lg border border-dashed border-gray-300 bg-white py-12 text-center text-sm text-gray-500"
          role="status"
        >
          {emptyLabel}
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50">
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                  {productColumnLabel}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                  {lastMessageColumnLabel}
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                  {statusColumnLabel}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {items.map((item) => (
                <tr key={item.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <Link href={item.href} className="block">
                      <div className="font-medium text-blue-600 hover:text-blue-800">
                        {item.subject}
                      </div>
                      <div className="mt-0.5 text-xs text-gray-500">
                        {item.productName}
                        <span className="ml-2 font-mono text-gray-400">{item.skuCode}</span>
                      </div>
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">{item.lastMessageAt}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span
                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                          item.status === "open"
                            ? "bg-green-100 text-green-800"
                            : "bg-gray-100 text-gray-600"
                        }`}
                      >
                        {item.statusLabel}
                      </span>
                      {item.unreadCount > 0 && (
                        <span
                          className="inline-flex items-center rounded-full bg-blue-600 px-2 py-0.5 text-xs font-semibold text-white"
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
      )}
    </section>
  );
}
