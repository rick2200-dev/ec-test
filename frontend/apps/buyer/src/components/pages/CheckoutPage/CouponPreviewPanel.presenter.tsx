import { formatCurrency } from "@ec-marketplace/ui-utils";
import type { CouponPreviewResult } from "@ec-marketplace/types";

export interface CouponPreviewPanelPresenterProps {
  code: string;
  onCodeChange: (value: string) => void;
  onApply: () => void;
  onClear: () => void;
  loading: boolean;
  /** When set, applied coupon details replace the apply button. */
  preview: CouponPreviewResult | null;
  /** Inline error (e.g. unknown code, min order not met). */
  error?: string;
  labels: {
    title: string;
    placeholder: string;
    apply: string;
    applying: string;
    clear: string;
    appliedDiscount: string;
  };
}

export function CouponPreviewPanelPresenter({
  code,
  onCodeChange,
  onApply,
  onClear,
  loading,
  preview,
  error,
  labels,
}: CouponPreviewPanelPresenterProps) {
  return (
    <section className="rounded-lg border border-gray-200 bg-white p-6">
      <h2 className="text-base font-semibold text-gray-900">{labels.title}</h2>
      <div className="mt-3 flex items-center gap-2">
        <input
          type="text"
          value={code}
          onChange={(e) => onCodeChange(e.target.value)}
          placeholder={labels.placeholder}
          disabled={loading || preview !== null}
          className="block flex-1 rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50"
        />
        {preview ? (
          <button
            type="button"
            onClick={onClear}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {labels.clear}
          </button>
        ) : (
          <button
            type="button"
            onClick={onApply}
            disabled={loading || code.trim().length === 0}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? labels.applying : labels.apply}
          </button>
        )}
      </div>

      {preview && (
        <div className="mt-3 rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-800">
          <div className="font-medium">{preview.title}</div>
          {preview.description && <div className="text-xs">{preview.description}</div>}
          <div className="mt-1">
            {labels.appliedDiscount}: <strong>−{formatCurrency(preview.discount_amount)}</strong>
          </div>
        </div>
      )}

      {error && (
        <p role="alert" className="mt-2 text-sm text-red-600">
          {error}
        </p>
      )}
    </section>
  );
}
