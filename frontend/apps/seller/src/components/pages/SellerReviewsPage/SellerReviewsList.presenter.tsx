import type { Review } from "@ec-marketplace/types";

export interface SellerReviewsListPresenterProps {
  reviews: Review[];
  loading: boolean;
  loaded: boolean;
  loadingLabel: string;
  emptyLabel: string;
  noReplyBadge: string;
  hasReplyBadge: string;
  renderRow: (review: Review) => React.ReactNode;
  locale: string;
  columns: {
    product: string;
    rating: string;
    title: string;
    date: string;
    state: string;
    actions: string;
  };
}

export function SellerReviewsListPresenter({
  reviews,
  loading,
  loaded,
  loadingLabel,
  emptyLabel,
  noReplyBadge,
  hasReplyBadge,
  renderRow,
  locale,
  columns,
}: SellerReviewsListPresenterProps) {
  if (!loaded || loading) {
    return <p className="py-8 text-center text-sm text-text-secondary">{loadingLabel}</p>;
  }
  if (reviews.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-border py-8 text-center text-sm text-text-secondary">
        {emptyLabel}
      </p>
    );
  }

  const dtf = new Intl.DateTimeFormat(locale, { dateStyle: "medium" });

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-white shadow-sm">
      <table className="min-w-full divide-y divide-border text-sm">
        <thead className="bg-surface text-left text-xs font-medium uppercase tracking-wider text-text-secondary">
          <tr>
            <th className="px-4 py-3">{columns.product}</th>
            <th className="px-4 py-3">{columns.rating}</th>
            <th className="px-4 py-3">{columns.title}</th>
            <th className="px-4 py-3">{columns.date}</th>
            <th className="px-4 py-3">{columns.state}</th>
            <th className="px-4 py-3 text-right">{columns.actions}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {reviews.map((review) => (
            <tr key={review.id} className="align-top">
              <td className="px-4 py-3 text-text-primary">{review.product_name}</td>
              <td className="px-4 py-3 text-text-secondary">{"★".repeat(review.rating)}</td>
              <td className="px-4 py-3">
                <div className="font-medium text-text-primary">{review.title}</div>
                <p className="mt-1 whitespace-pre-wrap text-xs text-text-secondary">
                  {review.body}
                </p>
                {review.reply && (
                  <div className="mt-2 rounded-md border-l-2 border-accent bg-surface px-3 py-2 text-xs text-text-primary">
                    <p className="whitespace-pre-wrap">{review.reply.body}</p>
                  </div>
                )}
              </td>
              <td className="px-4 py-3 text-xs text-text-secondary">
                {dtf.format(new Date(review.created_at))}
              </td>
              <td className="px-4 py-3">
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                    review.reply ? "bg-green-100 text-green-800" : "bg-yellow-100 text-yellow-800"
                  }`}
                >
                  {review.reply ? hasReplyBadge : noReplyBadge}
                </span>
              </td>
              <td className="px-4 py-3 text-right">{renderRow(review)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
