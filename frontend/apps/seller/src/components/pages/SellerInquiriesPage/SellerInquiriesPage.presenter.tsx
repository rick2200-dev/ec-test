import type { ReactNode } from "react";

export interface SellerInquiriesPagePresenterProps {
  listSlot: ReactNode;
  errorSlot?: ReactNode;
}

export function SellerInquiriesPagePresenter({
  listSlot,
  errorSlot,
}: SellerInquiriesPagePresenterProps) {
  return (
    <div className="space-y-4">
      {errorSlot}
      {listSlot}
    </div>
  );
}
