"use client";

import { type FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";

import { ApiError, requestOrderCancellation } from "@/lib/api";
import { CancelOrderButtonPresenter } from "./CancelOrderButton.presenter";

export interface CancelOrderButtonProps {
  orderId: string;
}

/**
 * CancelOrderButton is rendered inside the buyer order detail page when
 * the order is still cancellable and no pending/terminal-approved request
 * already exists. Its responsibilities:
 *
 *  1. Collect a free-text reason in a modal.
 *  2. POST it to the gateway (which proxies to order-svc).
 *  3. On success, refresh the server component via `router.refresh()` so
 *     the page re-renders with the new cancellation-request status.
 *
 * Semantic error codes defined in
 * `backend/services/order/internal/cancellation/errors.go` are matched
 * on `ApiError.code` rather than message content.
 */
export default function CancelOrderButton({ orderId }: CancelOrderButtonProps) {
  const t = useTranslations("orders.cancellation");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setReason("");
    setError(null);
    setSubmitting(false);
  };

  const handleOpen = () => {
    reset();
    setOpen(true);
  };

  const handleClose = () => {
    if (submitting) return;
    setOpen(false);
    reset();
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submitting) return;

    const trimmed = reason.trim();
    if (!trimmed) {
      setError(t("errorEmpty"));
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      await requestOrderCancellation(orderId, trimmed);
      setOpen(false);
      reset();
      router.refresh();
    } catch (err) {
      if (err instanceof ApiError) {
        // Prefer semantic code over status — two different 409s need
        // different UI copy.
        if (err.code === "ORDER_NOT_CANCELLABLE") {
          setError(t("errorNotCancellable"));
        } else if (err.code === "CANCELLATION_REQUEST_ALREADY_EXISTS") {
          setError(t("errorAlreadyExists"));
        } else {
          setError(err.message || t("errorGeneric"));
        }
      } else if (err instanceof Error) {
        setError(err.message || t("errorGeneric"));
      } else {
        setError(t("errorGeneric"));
      }
      setSubmitting(false);
    }
  };

  return (
    <CancelOrderButtonPresenter
      triggerLabel={t("trigger")}
      modalTitle={t("modalTitle")}
      modalDescription={t("modalDescription")}
      reasonLabel={t("reasonLabel")}
      reasonPlaceholder={t("reasonPlaceholder")}
      reasonValue={reason}
      onReasonChange={setReason}
      open={open}
      onOpen={handleOpen}
      onClose={handleClose}
      onSubmit={handleSubmit}
      submitLabel={t("submit")}
      submittingLabel={t("submitting")}
      cancelLabel={t("cancel")}
      submitting={submitting}
      error={error}
    />
  );
}
