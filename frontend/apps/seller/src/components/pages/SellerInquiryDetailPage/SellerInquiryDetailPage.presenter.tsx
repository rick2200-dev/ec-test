import type { ReactNode } from "react";

export interface SellerInquiryDetailPagePresenterProps {
  threadSlot: ReactNode;
}

export function SellerInquiryDetailPagePresenter({
  threadSlot,
}: SellerInquiryDetailPagePresenterProps) {
  return <div className="max-w-3xl">{threadSlot}</div>;
}
