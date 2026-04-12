"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";

import type { CancellationRequest, CancellationRequestStatus } from "@ec-marketplace/types";

import {
  ApiError,
  approveCancellationRequest,
  listSellerCancellationRequests,
  rejectCancellationRequest,
} from "@/lib/api";

import {
  CancellationActionModalPresenter,
  CancellationRequestsPagePresenter,
  type CancellationRequestRow,
} from "./CancellationRequestsPage.presenter";

const FILTER_VALUES: CancellationRequestStatus[] = ["pending", "approved", "rejected", "failed"];

type ActionKind = "approve" | "reject";

/**
 * Seller cancellation-requests dashboard. Lists pending requests (or
 * filters by any terminal status), exposes approve/reject actions, and
 * handles semantic error codes returned by the order service.
 *
 * Approving triggers Stripe refund + transfer reversal + inventory
 * release on the backend, so a single approve click is the most
 * load-bearing action in this page. We explicitly confirm via a modal
 * with an optional comment before firing the request.
 */
export default function CancellationRequestsPage() {
  const t = useTranslations("cancellationRequests");
  const locale = useLocale();
  const dateFormatter = new Intl.DateTimeFormat(locale === "ja" ? "ja-JP" : "en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

  const [filter, setFilter] = useState<CancellationRequestStatus>("pending");
  const [items, setItems] = useState<CancellationRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null);

  // Modal state — one at a time; we store which kind is open plus the
  // request id it's targeting so the submit handler knows what to do.
  const [modalKind, setModalKind] = useState<ActionKind | null>(null);
  const [modalRequestId, setModalRequestId] = useState<string | null>(null);
  const [comment, setComment] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [modalError, setModalError] = useState<string | null>(null);

  const load = useCallback(
    async (status: CancellationRequestStatus) => {
      setLoading(true);
      setListError(null);
      try {
        const res = await listSellerCancellationRequests({ status, limit: 50 });
        setItems(res.items ?? []);
      } catch (err) {
        setItems([]);
        setListError(err instanceof Error ? err.message : t("errors.loadFailed"));
      } finally {
        setLoading(false);
      }
    },
    [t]
  );

  useEffect(() => {
    void load(filter);
  }, [filter, load]);

  const openModal = (kind: ActionKind, id: string) => {
    setModalKind(kind);
    setModalRequestId(id);
    setComment("");
    setModalError(null);
  };

  const closeModal = () => {
    if (submitting) return;
    setModalKind(null);
    setModalRequestId(null);
    setComment("");
    setModalError(null);
  };

  const handleModalSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting || !modalKind || !modalRequestId) return;

    const trimmed = comment.trim();
    if (modalKind === "reject" && !trimmed) {
      setModalError(t("rejectModal.commentRequired"));
      return;
    }

    setSubmitting(true);
    setModalError(null);
    try {
      if (modalKind === "approve") {
        await approveCancellationRequest(modalRequestId, trimmed || undefined);
      } else {
        await rejectCancellationRequest(modalRequestId, trimmed);
      }
      setModalKind(null);
      setModalRequestId(null);
      setComment("");
      void load(filter);
    } catch (err) {
      setModalError(buildErrorMessage(err, modalKind, t));
      setSubmitting(false);
      return;
    }
    setSubmitting(false);
  };

  const rows: CancellationRequestRow[] = items.map((r) => ({
    id: r.id,
    orderId: r.order_id,
    reason: r.reason,
    requestedAtLabel: dateFormatter.format(new Date(r.created_at)),
    status: r.status,
    statusLabel: t(`statusLabel.${r.status}`),
  }));

  const filterOptions = FILTER_VALUES.map((value) => ({
    value,
    label: t(`filter.${value}`),
  }));

  const modalSlot =
    modalKind === "approve" ? (
      <CancellationActionModalPresenter
        open
        tone="approve"
        title={t("approveModal.title")}
        description={t("approveModal.description")}
        commentLabel={t("approveModal.commentLabel")}
        commentPlaceholder={t("approveModal.commentPlaceholder")}
        commentValue={comment}
        onCommentChange={setComment}
        commentRequired={false}
        onClose={closeModal}
        onSubmit={handleModalSubmit}
        submitLabel={t("approveModal.submit")}
        submittingLabel={t("approveModal.submitting")}
        cancelLabel={t("approveModal.cancel")}
        submitting={submitting}
        error={modalError}
      />
    ) : modalKind === "reject" ? (
      <CancellationActionModalPresenter
        open
        tone="reject"
        title={t("rejectModal.title")}
        description={t("rejectModal.description")}
        commentLabel={t("rejectModal.commentLabel")}
        commentPlaceholder={t("rejectModal.commentPlaceholder")}
        commentValue={comment}
        onCommentChange={setComment}
        commentRequired
        onClose={closeModal}
        onSubmit={handleModalSubmit}
        submitLabel={t("rejectModal.submit")}
        submittingLabel={t("rejectModal.submitting")}
        cancelLabel={t("rejectModal.cancel")}
        submitting={submitting}
        error={modalError}
      />
    ) : undefined;

  return (
    <CancellationRequestsPagePresenter
      title={t("title")}
      description={t("description")}
      filterLabel={t("filterLabel")}
      filterOptions={filterOptions}
      selectedFilter={filter}
      onFilterChange={setFilter}
      columns={{
        orderId: t("columns.orderId"),
        reason: t("columns.reason"),
        requestedAt: t("columns.requestedAt"),
        status: t("columns.status"),
        actions: t("columns.actions"),
      }}
      rows={rows}
      loading={loading}
      loadingLabel={t("loading")}
      emptyLabel={t("empty")}
      error={listError}
      approveLabel={t("action.approve")}
      rejectLabel={t("action.reject")}
      onApprove={(id) => openModal("approve", id)}
      onReject={(id) => openModal("reject", id)}
      modalSlot={modalSlot}
    />
  );
}

/**
 * Build a user-facing error message for an approve/reject failure.
 * Switches on semantic `ApiError.code` values (see
 * `backend/services/order/internal/cancellation/errors.go`) so copy
 * can stay decoupled from backend messages.
 */
function buildErrorMessage(
  err: unknown,
  action: ActionKind,
  t: ReturnType<typeof useTranslations>
): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "REFUND_FAILED":
        return t("errors.refundFailed");
      case "TRANSFER_REVERSAL_FAILED":
        return t("errors.transferReversalFailed");
    }
    if (err.message) return err.message;
  }
  if (err instanceof Error && err.message) return err.message;
  return action === "approve" ? t("errors.approveFailed") : t("errors.rejectFailed");
}
