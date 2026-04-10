import type { Meta, StoryObj } from "@storybook/react";
import { RecommendationsPresenter } from "./Recommendations.presenter";

const meta: Meta<typeof RecommendationsPresenter> = {
  title: "Buyer/Recommendations",
  component: RecommendationsPresenter,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof RecommendationsPresenter>;

const sampleItems = [
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

export const Loaded: Story = {
  args: {
    title: "Recommended for you",
    loading: false,
    items: sampleItems,
  },
};

export const Loading: Story = {
  args: {
    title: "Recommended for you",
    loading: true,
    items: [],
  },
};

export const EmptyHidden: Story = {
  args: {
    title: "Recommended for you",
    loading: false,
    items: [],
  },
  parameters: {
    docs: {
      description: {
        story: "When loading is false and items are empty, the section renders nothing.",
      },
    },
  },
};
