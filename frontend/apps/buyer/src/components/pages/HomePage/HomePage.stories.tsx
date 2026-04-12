import type { Meta, StoryObj } from "@storybook/react";
import { HomePagePresenter } from "./HomePage.presenter";
import { RecommendationsPresenter } from "../../Recommendations/Recommendations.presenter";

const meta: Meta<typeof HomePagePresenter> = {
  title: "Buyer/Pages/HomePage",
  component: HomePagePresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof HomePagePresenter>;

const sampleHero = {
  titleLines: ["こだわりの商品を", "あなたの手に"],
  subtitle: "全国のセラーから選び抜かれた商品を",
  ctaHref: "/products",
  ctaLabel: "商品を見る",
};

const sampleCategories = [
  {
    id: "c1",
    href: "/products?category=electronics",
    name: "家電",
    iconPath:
      "M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3",
    viewProductsLabel: "商品を見る",
  },
  {
    id: "c2",
    href: "/products?category=fashion",
    name: "ファッション",
    iconPath:
      "M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5",
    viewProductsLabel: "商品を見る",
  },
  {
    id: "c3",
    href: "/products?category=handmade",
    name: "ハンドメイド",
    iconPath:
      "M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5",
    viewProductsLabel: "商品を見る",
  },
];

const sampleProducts = [
  {
    id: "p1",
    href: "/products/wireless-headphones",
    name: "Wireless Headphones",
    sellerName: "Acme Audio",
    priceLabel: "¥12,800",
  },
  {
    id: "p2",
    href: "/products/leather-bag",
    name: "Leather Tote Bag",
    sellerName: "Studio Leather",
    priceLabel: "¥24,000",
  },
  {
    id: "p3",
    href: "/products/ceramic-mug",
    name: "Handmade Ceramic Mug",
    sellerName: "Pottery Lab",
    priceLabel: "¥3,200",
  },
  {
    id: "p4",
    href: "/products/wool-scarf",
    name: "Wool Scarf",
    sellerName: "Knit Studio",
    priceLabel: "¥6,800",
  },
  {
    id: "p5",
    href: "/products/desk-lamp",
    name: "Brass Desk Lamp",
    sellerName: "Lumen Works",
    priceLabel: "¥18,000",
  },
  {
    id: "p6",
    href: "/products/notebook",
    name: "Hardcover Notebook",
    sellerName: "Paper Goods",
    priceLabel: "¥1,500",
  },
];

export const Default: Story = {
  args: {
    hero: sampleHero,
    categoriesSection: { title: "カテゴリ", items: sampleCategories },
    popularSection: {
      title: "人気の商品",
      showAllHref: "/products",
      showAllLabel: "すべて見る",
      products: sampleProducts,
    },
    recommendationsSlot: (
      <RecommendationsPresenter title="あなたへのおすすめ" loading={false} items={sampleProducts} />
    ),
  },
};

export const RecommendationsLoading: Story = {
  args: {
    ...Default.args,
    recommendationsSlot: (
      <RecommendationsPresenter title="あなたへのおすすめ" loading={true} items={[]} />
    ),
  },
};

export const NoRecommendations: Story = {
  args: {
    ...Default.args,
    recommendationsSlot: undefined,
  },
};
