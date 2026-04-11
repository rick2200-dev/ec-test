"use client";

import { type FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { createInquiry } from "@/lib/api";
import { StartInquiryButtonPresenter } from "./StartInquiryButton.presenter";

export interface StartInquiryButtonProps {
  sellerId: string;
  skuId: string;
  productName: string;
  skuCode: string;
}

export default function StartInquiryButton({
  sellerId,
  skuId,
  productName,
  skuCode,
}: StartInquiryButtonProps) {
  const t = useTranslations("inquiries");
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reset = () => {
    setSubject("");
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
    setSubmitting(true);
    setError(null);
    try {
      const created = await createInquiry({
        seller_id: sellerId,
        sku_id: skuId,
        subject: subject.trim(),
        initial_body: body.trim(),
      });
      setOpen(false);
      reset();
      router.push(`/inquiries/${created.id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : "";
      // Forbidden means "not purchased" — show the friendlier copy.
      if (message.toLowerCase().includes("forbidden") || message.includes("403")) {
        setError(t("errorForbidden"));
      } else {
        setError(message || t("errorGeneric"));
      }
      setSubmitting(false);
    }
  };

  return (
    <StartInquiryButtonPresenter
      triggerLabel={t("new")}
      modalTitle={t("new")}
      productName={productName}
      skuCode={skuCode}
      subjectLabel={t("subject")}
      subjectPlaceholder={t("subjectPlaceholder")}
      subjectValue={subject}
      onSubjectChange={setSubject}
      bodyLabel={t("initialBody")}
      bodyPlaceholder={t("bodyPlaceholder")}
      bodyValue={body}
      onBodyChange={setBody}
      open={open}
      onOpen={handleOpen}
      onClose={handleClose}
      onSubmit={handleSubmit}
      submitLabel={t("send")}
      submittingLabel={t("sending")}
      cancelLabel={t("cancel")}
      submitting={submitting}
      error={error}
    />
  );
}
