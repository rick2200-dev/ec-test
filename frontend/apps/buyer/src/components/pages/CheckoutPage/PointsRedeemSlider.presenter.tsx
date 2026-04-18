import { formatCurrency } from "@ec-marketplace/ui-utils";

export interface PointsRedeemSliderPresenterProps {
  /** Spendable balance (already accounts for pending reservations). */
  spendable: number;
  /** Cap above which points cannot be redeemed (subtotal − discount). */
  maxRedeemable: number;
  value: number;
  onChange: (value: number) => void;
  labels: {
    title: string;
    spendableLabel: string;
    useLabel: string;
    noPoints: string;
  };
}

export function PointsRedeemSliderPresenter({
  spendable,
  maxRedeemable,
  value,
  onChange,
  labels,
}: PointsRedeemSliderPresenterProps) {
  const cap = Math.max(0, Math.min(spendable, maxRedeemable));
  if (spendable <= 0) {
    return (
      <section className="rounded-lg border border-gray-200 bg-white p-6">
        <h2 className="text-base font-semibold text-gray-900">{labels.title}</h2>
        <p className="mt-2 text-sm text-gray-500">{labels.noPoints}</p>
      </section>
    );
  }

  return (
    <section className="rounded-lg border border-gray-200 bg-white p-6">
      <div className="flex items-baseline justify-between">
        <h2 className="text-base font-semibold text-gray-900">{labels.title}</h2>
        <span className="text-sm text-gray-500">
          {labels.spendableLabel}: {spendable.toLocaleString()} pt
        </span>
      </div>
      <input
        type="range"
        min={0}
        max={cap}
        step={1}
        value={Math.min(value, cap)}
        onChange={(e) => onChange(Number(e.target.value))}
        className="mt-4 w-full"
        aria-label={labels.useLabel}
      />
      <div className="mt-2 flex items-center justify-between text-sm text-gray-700">
        <span>0</span>
        <span>
          {labels.useLabel}: <strong>{value.toLocaleString()} pt</strong> ( −{formatCurrency(value)}
          )
        </span>
        <span>{cap.toLocaleString()}</span>
      </div>
    </section>
  );
}
