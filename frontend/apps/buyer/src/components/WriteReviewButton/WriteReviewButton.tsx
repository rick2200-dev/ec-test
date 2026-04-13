"use client";

import { type FormEvent, useEffect, useState } from "react";
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
  const tAuth = useTranslations("auth");
  // Poll /auth/profile once on mount to decide whether to show the
  // modal trigger or a "log in to review" link. The endpoint is mounted
  // by the Auth0 SDK middleware and returns 200 w/ the user payload
  // when a session cookie is present, 204/401 otherwise.
  const [loggedIn, setLoggedIn] = useState<boolean | null>(null);
  useEffect(() => {
    let cancelled = false;
    fetch("/auth/profile", { credentials: "include" })
      .then((res) => {
        if (!cancelled) setLoggedIn(res.ok);
      })
      .catch(() => {
        if (!cancelled) setLoggedIn(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);
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

  // Guest OR still-checking: render the login-to-review link by default.
  // Showing the normal "write review" button while loggedIn === null would
  // let a guest open the modal, submit, and see a generic error as the BFF
  // returns 401. Tilting toward the login link keeps the failure mode
  // explicit. Once the probe resolves as `true` we swap to the modal.
  if (loggedIn !== true) {
    const returnTo = typeof window !== "undefined" ? window.location.pathname : "/";
    return (
      <a
        href={`/auth/login?returnTo=${encodeURIComponent(returnTo)}`}
        aria-busy={loggedIn === null}
        className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50"
      >
        {tAuth("loginToReview")}
      </a>
    );
  }

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
