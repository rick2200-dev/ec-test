export interface ReviewReplyFormPresenterProps {
  open: boolean;
  mode: "create" | "edit";
  body: string;
  onBodyChange: (value: string) => void;
  submitting: boolean;
  deleting: boolean;
  onSubmit: () => void;
  onDelete?: () => void;
  onClose: () => void;
  errorMessage?: string;
  labels: {
    createTitle: string;
    editTitle: string;
    bodyLabel: string;
    bodyPlaceholder: string;
    submit: string;
    submitting: string;
    cancel: string;
    delete: string;
    deleting: string;
  };
}

export function ReviewReplyFormPresenter({
  open,
  mode,
  body,
  onBodyChange,
  submitting,
  deleting,
  onSubmit,
  onDelete,
  onClose,
  errorMessage,
  labels,
}: ReviewReplyFormPresenterProps) {
  if (!open) return null;
  const title = mode === "edit" ? labels.editTitle : labels.createTitle;
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
        className="w-full max-w-lg rounded-lg bg-white p-6 shadow-xl"
      >
        <h2 className="text-lg font-semibold text-text-primary">{title}</h2>

        <div className="mt-4">
          <label htmlFor="reply-body" className="block text-sm font-medium text-text-primary">
            {labels.bodyLabel} <span className="text-danger">*</span>
          </label>
          <textarea
            id="reply-body"
            value={body}
            onChange={(e) => onBodyChange(e.target.value)}
            placeholder={labels.bodyPlaceholder}
            rows={4}
            required
            disabled={submitting || deleting}
            className="mt-1 block w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent disabled:bg-surface"
          />
        </div>

        {errorMessage && (
          <p role="alert" className="mt-2 text-sm text-danger">
            {errorMessage}
          </p>
        )}

        <div className="mt-6 flex items-center justify-between gap-3">
          <div>
            {mode === "edit" && onDelete && (
              <button
                type="button"
                onClick={onDelete}
                disabled={submitting || deleting}
                className="text-sm font-medium text-danger hover:text-red-700 disabled:opacity-50"
              >
                {deleting ? labels.deleting : labels.delete}
              </button>
            )}
          </div>
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={submitting || deleting}
              className="rounded-md border border-border px-4 py-2 text-sm font-medium text-text-primary hover:bg-surface-hover disabled:opacity-50"
            >
              {labels.cancel}
            </button>
            <button
              type="submit"
              disabled={submitting || deleting || body.trim().length === 0}
              className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-white hover:bg-accent-hover disabled:opacity-50"
            >
              {submitting ? labels.submitting : labels.submit}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}
