import type { Meta, StoryObj } from "@storybook/react";
import { ProductCardPresenter } from "./ProductCard.presenter";

const meta: Meta<typeof ProductCardPresenter> = {
  title: "Buyer/ProductCard",
  component: ProductCardPresenter,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <div style={{ width: 240 }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof ProductCardPresenter>;

export const Default: Story = {
  args: {
    href: "/products/wireless-headphones",
    name: "Wireless Headphones",
    sellerName: "Acme Audio",
    priceLabel: "¥12,800",
  },
};

export const LongName: Story = {
  args: {
    href: "/products/premium-headphones",
    name: "Premium Noise-Cancelling Over-Ear Wireless Bluetooth Headphones with 40-Hour Battery Life",
    sellerName: "Acme Audio",
    priceLabel: "¥24,800",
  },
};

export const WithoutSeller: Story = {
  args: {
    href: "/products/mystery-item",
    name: "Mystery Item",
    priceLabel: "¥980",
  },
};
