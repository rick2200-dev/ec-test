"use client";

import { type FormEvent } from "react";

export interface StartInquiryButtonPresenterProps {
  /** Label on the trigger button (e.g., "出品者に問い合わせる"). */
  triggerLabel: string;
  /** Title shown at the top of the modal. */
  modalTitle: string;
  /** Product name shown as a read-only header in the modal. */
  productName: string;
  skuCode: string;
  subjectLabel: string;
  subjectPlaceholder: string;
  subjectValue: string;
  onSubjectChange: (value: string) => void;
  bodyLabel: string;
  bodyPlaceholder: string;
  bodyValue: string;
  onBodyChange: (value: string) => void;
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

export function StartInquiryButtonPresenter({
  triggerLabel,
  modalTitle,
  productName,
  skuCode,
  subjectLabel,
  subjectPlaceholder,
  subjectValue,
  onSubjectChange,
  bodyLabel,
  bodyPlaceholder,
  bodyValue,
  onBodyChange,
  open,
  onOpen,
  onClose,
  onSubmit,
  submitLabel,
  submittingLabel,
  cancelLabel,
  submitting,
  error,
}: StartInquiryButtonPresenterProps) {
  return (
    <>
      <button
        type="button"
        onClick={onOpen}
        className="inline-flex items-center rounded-md border border-blue-600 px-3 py-1.5 text-sm font-medium text-blue-600 hover:bg-blue-50"
      >
        {triggerLabel}
      </button>

      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="inquiry-modal-title"
        >
          <form onSubmit={onSubmit} className="w-full max-w-lg rounded-lg bg-white shadow-xl">
            <header className="border-b border-gray-200 px-5 py-3">
              <h2 id="inquiry-modal-title" className="text-base font-semibold text-gray-900">
                {modalTitle}
              </h2>
              <p className="mt-1 text-xs text-gray-500">
                {productName}
                <span className="ml-2 font-mono text-gray-400">{skuCode}</span>
              </p>
            </header>

            <div className="space-y-3 px-5 py-4">
              <label className="block">
                <span className="text-sm font-medium text-gray-700">{subjectLabel}</span>
                <input
                  type="text"
                  value={subjectValue}
                  onChange={(e) => onSubjectChange(e.target.value)}
                  placeholder={subjectPlaceholder}
                  maxLength={255}
                  required
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </label>
              <label className="block">
                <span className="text-sm font-medium text-gray-700">{bodyLabel}</span>
                <textarea
                  value={bodyValue}
                  onChange={(e) => onBodyChange(e.target.value)}
                  rows={5}
                  maxLength={4000}
                  placeholder={bodyPlaceholder}
                  required
                  className="mt-1 w-full resize-none rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
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
                className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300"
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
