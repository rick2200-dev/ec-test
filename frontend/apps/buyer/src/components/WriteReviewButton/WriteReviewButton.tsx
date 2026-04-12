"use client";

import { type FormEvent, useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError, createReview } from "@/lib/api";
import { WriteReviewButtonPresenter } from "./WriteReviewButton.presenter";

export interface WriteReviewButtonProps {
  productId: string;
  productName: string;
  onSuccess?: () => void;
}

export default function WriteReviewButton({
  productId,
  productName,
  onSuccess,
}: WriteReviewButtonProps) {
  const t = useTranslations("reviews");
  const [open, setOpen] = useState(false);
  const [rating, setRating] = useState(0);
  const [ratingHover, setRatingHover] = useState<number | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setRating(0);
    setRatingHover(null);
    setTitle("");
    setBody("");
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

    if (rating === 0) {
      setError(t("errorNoRating"));
      return;
    }

    const trimmedTitle = title.trim();
    const trimmedBody = body.trim();
    if (!trimmedTitle || !trimmedBody) {
      setError(t("errorEmpty"));
      return;
    }

    setSubmitting(true);
    setError(null);
    try {
      await createReview({
        product_id: productId,
        rating,
        title: trimmedTitle,
        body: trimmedBody,
      });
      setOpen(false);
      reset();
      onSuccess?.();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "PURCHASE_REQUIRED" || err.status === 403) {
          setError(t("errorPurchaseRequired"));
        } else if (err.code === "ALREADY_REVIEWED" || err.status === 409) {
          setError(t("errorAlreadyReviewed"));
        } else {
          setError(t("errorGeneric"));
        }
      } else {
        setError(t("errorGeneric"));
      }
      setSubmitting(false);
    }
  };

  return (
    <WriteReviewButtonPresenter
      triggerLabel={t("write")}
      modalTitle={t("writeTitle")}
      productName={productName}
      ratingLabel={t("ratingLabel")}
      ratingValue={rating}
      ratingHoverValue={ratingHover}
      onRatingChange={setRating}
      onRatingHover={setRatingHover}
      titleLabel={t("titleLabel")}
      titlePlaceholder={t("titlePlaceholder")}
      titleValue={title}
      onTitleChange={setTitle}
      bodyLabel={t("bodyLabel")}
      bodyPlaceholder={t("bodyPlaceholder")}
      bodyValue={body}
      onBodyChange={setBody}
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
