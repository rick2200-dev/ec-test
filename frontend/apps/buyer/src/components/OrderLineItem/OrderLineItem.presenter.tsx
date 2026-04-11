import type { ReactNode } from "react";
import Link from "next/link";
import Image from "next/image";

export interface OrderLineItemPresenterProps {
  productName: string;
  skuCode: string;
  quantity: number;
  unitPriceLabel: string;
  lineTotalLabel: string;
  /** Current image URL from catalog, or empty string for no image. */
  imageUrl: string;
  /** Destination for the product-page link, or empty string if linking is disabled. */
  productHref: string;
  /** When true, the row renders a "deleted" badge and suppresses the link. */
  isDeleted: boolean;
  /** Localized labels injected by the container. */
  labels: {
    quantity: string;
    deletedBadge: string;
    imageMissing: string;
  };
  /** Optional trailing action (e.g. a StartInquiryButton). Rendered
   *  right-aligned beneath the price. Omitted when undefined. */
  actionSlot?: ReactNode;
}

/**
 * OrderLineItemPresenter renders one row in the order-detail page's item list.
 *
 * Variants:
 *  - live + image: linked row with <img>
 *  - live + no image: linked row with placeholder square
 *  - deleted / archived: non-linked row with "deleted" badge and placeholder
 */
export function OrderLineItemPresenter({
  productName,
  skuCode,
  quantity,
  unitPriceLabel,
  lineTotalLabel,
  imageUrl,
  productHref,
  isDeleted,
  labels,
  actionSlot,
}: OrderLineItemPresenterProps) {
  const imageBlock = imageUrl ? (
    <Image
      src={imageUrl}
      alt={productName}
      width={80}
      height={80}
      className="h-20 w-20 rounded-md object-cover"
      unoptimized
    />
  ) : (
    <div className="flex h-20 w-20 items-center justify-center rounded-md bg-gray-100 text-xs text-gray-400">
      <span>{labels.imageMissing}</span>
    </div>
  );

  const nameBlock = (
    <div className="flex items-start gap-2">
      <h3 className="text-sm font-medium text-gray-900 line-clamp-2">{productName}</h3>
      {isDeleted && (
        <span className="inline-flex items-center rounded-full bg-gray-200 px-2 py-0.5 text-xs font-medium text-gray-700">
          {labels.deletedBadge}
        </span>
      )}
    </div>
  );

  const content = (
    <div className="flex items-center gap-4">
      {imageBlock}
      <div className="flex-1 min-w-0">
        {nameBlock}
        <p className="mt-1 text-xs text-gray-500">{skuCode}</p>
        <p className="mt-1 text-xs text-gray-500">
          {labels.quantity}: {quantity} × {unitPriceLabel}
        </p>
      </div>
      <div className="shrink-0 text-right">
        <p className="text-sm font-semibold text-gray-900">{lineTotalLabel}</p>
        {actionSlot && <div className="mt-2">{actionSlot}</div>}
      </div>
    </div>
  );

  if (isDeleted || !productHref) {
    return <li className="py-4">{content}</li>;
  }

  return (
    <li className="py-4">
      <Link
        href={productHref}
        className="block rounded-md hover:bg-gray-50 -mx-2 px-2 transition-colors"
      >
        {content}
      </Link>
    </li>
  );
}
