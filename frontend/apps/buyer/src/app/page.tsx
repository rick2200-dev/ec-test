import { products, categories, formatPrice, getLowestPrice, getSellerById } from "@/lib/mock-data";
import { getTranslations } from "next-intl/server";
import { Recommendations } from "@/components/Recommendations";
import {
  HomePagePresenter,
  type HomePageCategoryItem,
  type HomePageProductItem,
} from "@/components/pages/HomePage/HomePage.presenter";

const categoryIcons: Record<string, string> = {
  electronics:
    "M 9 3 v 2 m 6 -2 v 2 M 9 19 v 2 m 6 -2 v 2 M 5 9 H 3 m 2 6 H 3 m 18 -6 h -2 m 2 6 h -2 M 7 19 h 10 a 2 2 0 0 0 2 -2 V 7 a 2 2 0 0 0 -2 -2 H 7 a 2 2 0 0 0 -2 2 v 10 a 2 2 0 0 0 2 2 Z m 4 -10 v 4 h 4",
  fashion:
    "M 15.75 10.5 l 4.72 -4.72 a 0.75 0.75 0 0 0 -1.28 -0.53 l -4.72 4.72 M 12 18.75 a 6 6 0 0 0 6 -6 v -1.5 m -6 7.5 a 6 6 0 0 1 -6 -6 v -1.5 m 6 7.5 v 3.75 m -3.75 0 h 7.5",
  handmade:
    "M 9.53 16.122 a 3 3 0 0 0 -5.78 1.128 2.25 2.25 0 0 1 -2.4 2.245 4.5 4.5 0 0 0 8.4 -2.245 c 0 -.399 -.078 -.78 -.22 -1.128 Z m 0 0 a 15.998 15.998 0 0 0 3.388 -1.62 m -5.043 -.025 a 15.994 15.994 0 0 1 1.622 -3.395 m 3.42 3.42 a 15.995 15.995 0 0 0 4.764 -4.648 l 3.876 -5.814 a 1.151 1.151 0 0 0 -1.597 -1.597 L 14.146 6.32 a 15.996 15.996 0 0 0 -4.649 4.763 m 3.42 3.42 a 6.776 6.776 0 0 0 -3.42 -3.42",
};

const fallbackCategoryIcon =
  "M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z";

export default async function HomePage() {
  const t = await getTranslations();
  const featuredProducts = products.filter((p) => p.status === "active").slice(0, 6);
  const heroLines = t("hero.title").split("\n");

  const categoryItems: HomePageCategoryItem[] = categories.map((category) => ({
    id: category.id,
    href: `/products?category=${category.slug}`,
    name: category.name,
    iconPath: categoryIcons[category.slug] ?? fallbackCategoryIcon,
    viewProductsLabel: t("product.viewProducts"),
  }));

  const popularProducts: HomePageProductItem[] = featuredProducts.map((product) => {
    const seller = getSellerById(product.seller_id);
    return {
      id: product.id,
      href: `/products/${product.slug}`,
      name: product.name,
      sellerName: seller?.name,
      priceLabel: formatPrice(getLowestPrice(product.id)),
    };
  });

  return (
    <HomePagePresenter
      hero={{
        titleLines: heroLines,
        subtitle: t("hero.subtitle"),
        ctaHref: "/products",
        ctaLabel: t("hero.cta"),
      }}
      categoriesSection={{
        title: t("nav.categories"),
        items: categoryItems,
      }}
      popularSection={{
        title: t("product.popularProducts"),
        showAllHref: "/products",
        showAllLabel: t("common.showAll"),
        products: popularProducts,
      }}
      recommendationsSlot={<Recommendations type="popular" title={t("product.recommendations")} />}
    />
  );
}
