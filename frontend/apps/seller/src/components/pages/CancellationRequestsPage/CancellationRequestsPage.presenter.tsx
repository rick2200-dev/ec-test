"use client";

import type { FormEvent, ReactNode } from "react";
import type { CancellationRequestStatus } from "@ec-marketplace/types";

/**
 * Pure presenter for the seller CancellationRequestsPage. Renders the
 * status filter, the table of requests, and (conditionally) an
 * approve/reject modal. The container owns all state + data-fetching.
 */

export interface CancellationRequestRow {
  id: string;
  orderId: string;
  reason: string;
  requestedAtLabel: string;
  status: CancellationRequestStatus;
  statusLabel: string;
}

export interface CancellationRequestsPagePresenterProps {
  title: string;
  description: string;

  filterLabel: string;
  filterOptions: Array<{ value: CancellationRequestStatus; label: string }>;
  selectedFilter: CancellationRequestStatus;
  onFilterChange: (value: CancellationRequestStatus) => void;

  columns: {
    orderId: string;
    reason: string;
    requestedAt: string;
    status: string;
    actions: string;
  };

  rows: CancellationRequestRow[];
  loading: boolean;
  loadingLabel: string;
  emptyLabel: string;
  error?: string | null;

  approveLabel: string;
  rejectLabel: string;
  onApprove: (id: string) => void;
  onReject: (id: string) => void;

  modalSlot?: ReactNode;
}

export interface CancellationActionModalPresenterProps {
  title: string;
  description: string;
  commentLabel: string;
  commentPlaceholder: string;
  commentValue: string;
  onCommentChange: (value: string) => void;
  commentRequired: boolean;
  open: boolean;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitLabel: string;
  submittingLabel: string;
  cancelLabel: string;
  submitting: boolean;
  error: string | null;
  /** "approve" tones the submit button green; "reject" tones it red. */
  tone: "approve" | "reject";
}

export function CancellationActionModalPresenter({
  title,
  description,
  commentLabel,
  commentPlaceholder,
  commentValue,
  onCommentChange,
  commentRequired,
  open,
  onClose,
  onSubmit,
  submitLabel,
  submittingLabel,
  cancelLabel,
  submitting,
  error,
  tone,
}: CancellationActionModalPresenterProps) {
  if (!open) return null;
  const submitClass =
    tone === "approve"
      ? "bg-emerald-600 hover:bg-emerald-700 disabled:bg-emerald-300"
      : "bg-red-600 hover:bg-red-700 disabled:bg-red-300";
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4"
      role="dialog"
      aria-modal="true"
    >
      <form onSubmit={onSubmit} className="w-full max-w-lg rounded-lg bg-white shadow-xl">
        <header className="border-b border-gray-200 px-5 py-3">
          <h2 className="text-base font-semibold text-gray-900">{title}</h2>
          <p className="mt-1 text-xs text-gray-500">{description}</p>
        </header>
        <div className="space-y-3 px-5 py-4">
          <label className="block">
            <span className="text-sm font-medium text-gray-700">{commentLabel}</span>
            <textarea
              value={commentValue}
              onChange={(e) => onCommentChange(e.target.value)}
              rows={4}
              maxLength={2000}
              placeholder={commentPlaceholder}
              required={commentRequired}
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
            className={`rounded-md px-3 py-1.5 text-sm font-semibold text-white disabled:cursor-not-allowed ${submitClass}`}
          >
            {submitting ? submittingLabel : submitLabel}
          </button>
        </footer>
      </form>
    </div>
  );
}

export function CancellationRequestsPagePresenter({
  title,
  description,
  filterLabel,
  filterOptions,
  selectedFilter,
  onFilterChange,
  columns,
  rows,
  loading,
  loadingLabel,
  emptyLabel,
  error,
  approveLabel,
  rejectLabel,
  onApprove,
  onReject,
  modalSlot,
}: CancellationRequestsPagePresenterProps) {
  return (
    <div className="space-y-4">
      <header>
        <h1 className="text-2xl font-bold text-gray-900">{title}</h1>
        <p className="mt-1 text-sm text-gray-600">{description}</p>
      </header>

      {error && (
        <div
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          role="alert"
        >
          {error}
        </div>
      )}

      <div className="flex items-center gap-3">
        <label className="text-sm font-medium text-gray-700" htmlFor="cancellation-status-filter">
          {filterLabel}
        </label>
        <select
          id="cancellation-status-filter"
          value={selectedFilter}
          onChange={(e) => onFilterChange(e.target.value as CancellationRequestStatus)}
          className="rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          {filterOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <section className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-semibold text-gray-700">
                {columns.orderId}
              </th>
              <th className="px-4 py-2 text-left text-xs font-semibold text-gray-700">
                {columns.reason}
              </th>
              <th className="px-4 py-2 text-left text-xs font-semibold text-gray-700">
                {columns.requestedAt}
              </th>
              <th className="px-4 py-2 text-left text-xs font-semibold text-gray-700">
                {columns.status}
              </th>
              <th className="px-4 py-2 text-right text-xs font-semibold text-gray-700">
                {columns.actions}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {loading ? (
              <tr>
                <td colSpan={5} className="px-4 py-6 text-center text-sm text-gray-500">
                  {loadingLabel}
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-6 text-center text-sm text-gray-500">
                  {emptyLabel}
                </td>
              </tr>
            ) : (
              rows.map((row) => (
                <tr key={row.id}>
                  <td className="px-4 py-3 font-mono text-xs text-gray-700">{row.orderId}</td>
                  <td className="px-4 py-3 text-sm text-gray-900">
                    <span className="line-clamp-2">{row.reason}</span>
                  </td>
                  <td className="px-4 py-3 text-xs text-gray-500">{row.requestedAtLabel}</td>
                  <td className="px-4 py-3 text-xs">
                    <span
                      className={`inline-flex items-center rounded-full px-2 py-0.5 font-medium ${statusChipClass(row.status)}`}
                    >
                      {row.statusLabel}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right text-sm">
                    {row.status === "pending" ? (
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          onClick={() => onApprove(row.id)}
                          className="rounded-md bg-emerald-600 px-3 py-1 text-xs font-semibold text-white hover:bg-emerald-700"
                        >
                          {approveLabel}
                        </button>
                        <button
                          type="button"
                          onClick={() => onReject(row.id)}
                          className="rounded-md bg-white px-3 py-1 text-xs font-semibold text-red-600 ring-1 ring-inset ring-red-600 hover:bg-red-50"
                        >
                          {rejectLabel}
                        </button>
                      </div>
                    ) : (
                      <span className="text-xs text-gray-400">—</span>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </section>

      {modalSlot}
    </div>
  );
}

function statusChipClass(status: CancellationRequestStatus): string {
  switch (status) {
    case "pending":
      return "bg-amber-100 text-amber-800";
    case "approved":
      return "bg-emerald-100 text-emerald-800";
    case "rejected":
      return "bg-slate-100 text-slate-700";
    case "failed":
      return "bg-red-100 text-red-800";
  }
}
