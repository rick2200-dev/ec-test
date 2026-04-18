export interface RegisterTrackingModalPresenterProps {
  open: boolean;
  carrier: string;
  trackingNumber: string;
  note: string;
  submitting: boolean;
  onCarrierChange: (value: string) => void;
  onTrackingNumberChange: (value: string) => void;
  onNoteChange: (value: string) => void;
  onSubmit: () => void;
  onClose: () => void;
  errorMessage?: string;
  labels: {
    title: string;
    carrier: string;
    trackingNumber: string;
    note: string;
    submit: string;
    submitting: string;
    cancel: string;
  };
}

export function RegisterTrackingModalPresenter({
  open,
  carrier,
  trackingNumber,
  note,
  submitting,
  onCarrierChange,
  onTrackingNumberChange,
  onNoteChange,
  onSubmit,
  onClose,
  errorMessage,
  labels,
}: RegisterTrackingModalPresenterProps) {
  if (!open) return null;
  return (
    <div
      role="dialog"
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        className="w-full max-w-md rounded-lg bg-white p-6 shadow-xl"
      >
        <h2 className="text-lg font-semibold text-text-primary">{labels.title}</h2>

        <div className="mt-4 space-y-4">
          <div>
            <label htmlFor="ship-carrier" className="block text-sm font-medium text-text-primary">
              {labels.carrier} <span className="text-danger">*</span>
            </label>
            <input
              id="ship-carrier"
              type="text"
              value={carrier}
              onChange={(e) => onCarrierChange(e.target.value)}
              required
              disabled={submitting}
              className="mt-1 block w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:bg-surface"
            />
          </div>
          <div>
            <label htmlFor="ship-tracking" className="block text-sm font-medium text-text-primary">
              {labels.trackingNumber} <span className="text-danger">*</span>
            </label>
            <input
              id="ship-tracking"
              type="text"
              value={trackingNumber}
              onChange={(e) => onTrackingNumberChange(e.target.value)}
              required
              disabled={submitting}
              className="mt-1 block w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:bg-surface"
            />
          </div>
          <div>
            <label htmlFor="ship-note" className="block text-sm font-medium text-text-primary">
              {labels.note}
            </label>
            <textarea
              id="ship-note"
              value={note}
              onChange={(e) => onNoteChange(e.target.value)}
              disabled={submitting}
              rows={2}
              className="mt-1 block w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:bg-surface"
            />
          </div>

          {errorMessage && (
            <p role="alert" className="text-sm text-danger">
              {errorMessage}
            </p>
          )}
        </div>

        <div className="mt-6 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={submitting}
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-text-primary hover:bg-surface-hover disabled:opacity-50"
          >
            {labels.cancel}
          </button>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-white hover:bg-accent-hover disabled:opacity-50"
          >
            {submitting ? labels.submitting : labels.submit}
          </button>
        </div>
      </form>
    </div>
  );
}
