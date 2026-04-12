import type { ReactNode } from "react";

export interface InquiriesPagePresenterProps {
  /** Rendered <InquiryList> or similar listing component. */
  listSlot: ReactNode;
  /** Optional error banner slot. */
  errorSlot?: ReactNode;
}

export function InquiriesPagePresenter({ listSlot, errorSlot }: InquiriesPagePresenterProps) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      {errorSlot}
      {listSlot}
    </div>
  );
}
