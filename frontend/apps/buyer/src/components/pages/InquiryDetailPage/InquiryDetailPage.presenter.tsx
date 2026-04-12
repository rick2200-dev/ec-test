import type { ReactNode } from "react";

export interface InquiryDetailPagePresenterProps {
  /** Rendered <InquiryThread> or loading / error fallback. */
  threadSlot: ReactNode;
}

export function InquiryDetailPagePresenter({ threadSlot }: InquiryDetailPagePresenterProps) {
  return <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:px-8">{threadSlot}</div>;
}
