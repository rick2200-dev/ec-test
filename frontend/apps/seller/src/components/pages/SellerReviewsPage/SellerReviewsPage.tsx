"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import {
  createReviewReply,
  deleteReviewReply,
  listSellerReviews,
  updateReviewReply,
} from "@/lib/api";
import type { Review } from "@ec-marketplace/types";
import { SellerReviewsListPresenter } from "./SellerReviewsList.presenter";
import { ReviewReplyFormPresenter } from "./ReviewReplyForm.presenter";

export default function SellerReviewsPage() {
  const t = useTranslations("reviews");
  const locale = useLocale();

  const [reviews, setReviews] = useState<Review[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [editingFor, setEditingFor] = useState<Review | null>(null);
  const [replyBody, setReplyBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const reload = async () => {
    try {
      const res = await listSellerReviews({ limit: 50 });
      setReviews(res.items ?? []);
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : t("errors.loadFailed"));
    }
  };

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await listSellerReviews({ limit: 50 });
        if (cancelled) return;
        setReviews(res.items ?? []);
      } catch (err) {
        if (cancelled) return;
        setLoadError(err instanceof Error ? err.message : t("errors.loadFailed"));
      } finally {
        if (!cancelled) setLoaded(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [t]);

  const openForm = (review: Review) => {
    setEditingFor(review);
    setReplyBody(review.reply?.body ?? "");
    setFormError(null);
  };

  const closeForm = () => {
    if (submitting || deleting) return;
    setEditingFor(null);
  };

  const handleSubmit = async () => {
    if (!editingFor || submitting) return;
    const body = replyBody.trim();
    if (!body) return;
    setSubmitting(true);
    setFormError(null);
    try {
      if (editingFor.reply) {
        await updateReviewReply(editingFor.id, body);
      } else {
        await createReviewReply(editingFor.id, body);
      }
      setEditingFor(null);
      await reload();
    } catch (err) {
      if (err instanceof ApiError) {
        setFormError(err.message || t("errors.replyFailed"));
      } else {
        setFormError(t("errors.replyFailed"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async () => {
    if (!editingFor || !editingFor.reply || deleting) return;
    if (typeof window !== "undefined" && !window.confirm(t("deleteConfirm"))) return;
    setDeleting(true);
    setFormError(null);
    try {
      await deleteReviewReply(editingFor.id);
      setEditingFor(null);
      await reload();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : t("errors.deleteFailed"));
    } finally {
      setDeleting(false);
    }
  };

  const mode: "create" | "edit" = editingFor?.reply ? "edit" : "create";

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("title")}</h2>
        <p className="mt-1 text-text-secondary">{t("description")}</p>
      </div>

      {loadError && (
        <div
          role="alert"
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger"
        >
          {loadError}
        </div>
      )}

      <SellerReviewsListPresenter
        reviews={reviews}
        loading={!loaded}
        loaded={loaded}
        loadingLabel={t("loading")}
        emptyLabel={t("empty")}
        noReplyBadge={t("badge.noReply")}
        hasReplyBadge={t("badge.replied")}
        locale={locale}
        columns={{
          product: t("columns.product"),
          rating: t("columns.rating"),
          title: t("columns.title"),
          date: t("columns.date"),
          state: t("columns.state"),
          actions: t("columns.actions"),
        }}
        renderRow={(review) => (
          <button
            type="button"
            onClick={() => openForm(review)}
            className="rounded-md border border-accent px-3 py-1.5 text-xs font-medium text-accent hover:bg-surface-hover"
          >
            {review.reply ? t("actions.edit") : t("actions.reply")}
          </button>
        )}
      />

      <ReviewReplyFormPresenter
        open={editingFor !== null}
        mode={mode}
        body={replyBody}
        onBodyChange={setReplyBody}
        submitting={submitting}
        deleting={deleting}
        onSubmit={handleSubmit}
        onDelete={mode === "edit" ? handleDelete : undefined}
        onClose={closeForm}
        errorMessage={formError ?? undefined}
        labels={{
          createTitle: t("form.createTitle"),
          editTitle: t("form.editTitle"),
          bodyLabel: t("form.bodyLabel"),
          bodyPlaceholder: t("form.bodyPlaceholder"),
          submit: t("form.submit"),
          submitting: t("form.submitting"),
          cancel: t("form.cancel"),
          delete: t("form.delete"),
          deleting: t("form.deleting"),
        }}
      />
    </div>
  );
}
