import { useTranslations } from "next-intl";
import type { ProductHit } from "@/lib/types";
import { PromotedBannerPresenter, type PromotedBannerItem } from "./PromotedBanner.presenter";

interface PromotedBannerProps {
  products: ProductHit[];
}

function formatPrice(amount: number, currency: string) {
  if (currency === "JPY") {
    return `\u00A5${amount.toLocaleString()}`;
  }
  return `${amount.toLocaleString()} ${currency}`;
}

export default function PromotedBanner({ products }: PromotedBannerProps) {
  const t = useTranslations();

  const items: PromotedBannerItem[] = products.map((p) => ({
    id: p.id,
    href: `/products/${p.slug}`,
    name: p.name,
    sellerName: p.seller_name,
    priceLabel: formatPrice(p.price_amount, p.price_currency),
  }));

  return <PromotedBannerPresenter items={items} sponsoredLabel={t("search.sponsored")} />;
}
