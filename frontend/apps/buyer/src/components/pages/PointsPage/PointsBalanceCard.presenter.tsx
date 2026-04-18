export interface PointsBalanceCardPresenterProps {
  spendable: number;
  pendingRedemption: number;
  lifetimeEarned: number;
  lifetimeRedeemed: number;
  labels: {
    spendable: string;
    pending: string;
    lifetimeEarned: string;
    lifetimeRedeemed: string;
    pointsUnit: string;
  };
}

export function PointsBalanceCardPresenter({
  spendable,
  pendingRedemption,
  lifetimeEarned,
  lifetimeRedeemed,
  labels,
}: PointsBalanceCardPresenterProps) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-6">
      <div className="text-center">
        <p className="text-sm font-medium text-gray-500">{labels.spendable}</p>
        <p className="mt-2 text-4xl font-bold text-blue-600">
          {spendable.toLocaleString()}
          <span className="ml-1 text-base font-normal text-gray-500">{labels.pointsUnit}</span>
        </p>
      </div>
      <dl className="mt-6 grid grid-cols-3 gap-4 border-t border-gray-200 pt-4 text-center text-sm">
        <div>
          <dt className="text-xs text-gray-500">{labels.pending}</dt>
          <dd className="mt-1 text-base font-semibold text-gray-900">
            {pendingRedemption.toLocaleString()}
          </dd>
        </div>
        <div>
          <dt className="text-xs text-gray-500">{labels.lifetimeEarned}</dt>
          <dd className="mt-1 text-base font-semibold text-gray-900">
            {lifetimeEarned.toLocaleString()}
          </dd>
        </div>
        <div>
          <dt className="text-xs text-gray-500">{labels.lifetimeRedeemed}</dt>
          <dd className="mt-1 text-base font-semibold text-gray-900">
            {lifetimeRedeemed.toLocaleString()}
          </dd>
        </div>
      </dl>
    </section>
  );
}
