import type { Meta, StoryObj } from "@storybook/react";
import { PromotedBannerPresenter } from "./PromotedBanner.presenter";

const meta: Meta<typeof PromotedBannerPresenter> = {
  title: "Buyer/PromotedBanner",
  component: PromotedBannerPresenter,
  parameters: { layout: "padded" },
};

export default meta;
type Story = StoryObj<typeof PromotedBannerPresenter>;

export const Default: Story = {
  args: {
    sponsoredLabel: "Sponsored",
    items: [
      {
        id: "p1",
        href: "/products/wireless-mouse",
        name: "Ergonomic Wireless Mouse",
        sellerName: "Acme Peripherals",
        priceLabel: "¥3,980",
      },
      {
        id: "p2",
        href: "/products/standing-desk",
        name: "Adjustable Standing Desk",
        sellerName: "Studio Furniture",
        priceLabel: "¥38,000",
      },
      {
        id: "p3",
        href: "/products/mechanical-keyboard",
        name: "Mechanical Keyboard",
        sellerName: "Keycap Lab",
        priceLabel: "¥18,500",
      },
    ],
  },
};

export const SingleItem: Story = {
  args: {
    sponsoredLabel: "Sponsored",
    items: [
      {
        id: "p1",
        href: "/products/wireless-mouse",
        name: "Ergonomic Wireless Mouse",
        sellerName: "Acme Peripherals",
        priceLabel: "¥3,980",
      },
    ],
  },
};

export const Empty: Story = {
  args: {
    sponsoredLabel: "Sponsored",
    items: [],
  },
};
