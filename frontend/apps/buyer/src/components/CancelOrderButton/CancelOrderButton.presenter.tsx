"use client";

import { type FormEvent } from "react";

/**
 * Pure presenter for the buyer-side cancel order flow: a trigger button
 * plus a modal asking for a reason. Kept separate from the container so
 * the stories / tests can render the UI without real API calls.
 */
export interface CancelOrderButtonPresenterProps {
  triggerLabel: string;
  modalTitle: string;
  modalDescription: string;
  reasonLabel: string;
  reasonPlaceholder: string;
  reasonValue: string;
  onReasonChange: (value: string) => void;
  open: boolean;
  onOpen: () => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitLabel: string;
  submittingLabel: string;
  cancelLabel: string;
  submitting: boolean;
  error: string | null;
}

export function CancelOrderButtonPresenter({
  triggerLabel,
  modalTitle,
  modalDescription,
  reasonLabel,
  reasonPlaceholder,
  reasonValue,
  onReasonChange,
  open,
  onOpen,
  onClose,
  onSubmit,
  submitLabel,
  submittingLabel,
  cancelLabel,
  submitting,
  error,
}: CancelOrderButtonPresenterProps) {
  return (
    <>
      <button
        type="button"
        onClick={onOpen}
        className="inline-flex items-center rounded-md border border-red-600 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50"
      >
        {triggerLabel}
      </button>

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="cancel-order-modal-title"
        >
          <form
            onSubmit={onSubmit}
            className="w-full max-w-lg rounded-lg bg-white shadow-xl"
          >
            <header className="border-b border-gray-200 px-5 py-3">
              <h2
                id="cancel-order-modal-title"
                className="text-base font-semibold text-gray-900"
              >
                {modalTitle}
              </h2>
              <p className="mt-1 text-xs text-gray-500">{modalDescription}</p>
            </header>

            <div className="space-y-3 px-5 py-4">
              <label className="block">
                <span className="text-sm font-medium text-gray-700">
                  {reasonLabel}
                </span>
                <textarea
                  value={reasonValue}
                  onChange={(e) => onReasonChange(e.target.value)}
                  rows={5}
                  maxLength={2000}
                  placeholder={reasonPlaceholder}
                  required
                  className="mt-1 w-full resize-none rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500"
                />
              </label>
              {error && (
                <p className="text-sm text-red-600" role="alert">
                  {error}
                </p>
              )}
            </div>

            <footer className="flex items-center justify-end gap-2 border-t border-gray-200 bg-gray-50 px-5 py-3">
              <button
                type="button"
                onClick={onClose}
                className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                {cancelLabel}
              </button>
              <button
                type="submit"
                disabled={submitting}
                className="rounded-md bg-red-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-red-700 disabled:cursor-not-allowed disabled:bg-red-300"
              >
                {submitting ? submittingLabel : submitLabel}
              </button>
            </footer>
          </form>
        </div>
      )}
    </>
  );
}
