import type { Meta, StoryObj } from "@storybook/react";
import ProductCard from "./ProductCard";

const meta: Meta<typeof ProductCard> = {
  title: "Buyer/ProductCard",
  component: ProductCard,
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
type Story = StoryObj<typeof ProductCard>;

const baseProduct = {
  id: "prod-1",
  tenant_id: "t1",
  seller_id: "seller-1",
  name: "Wireless Headphones",
  slug: "wireless-headphones",
  description: "High-quality wireless headphones with noise cancellation",
  status: "active" as const,
  created_at: "2025-01-01T00:00:00Z",
  updated_at: "2025-01-01T00:00:00Z",
};

export const Default: Story = {
  args: { product: baseProduct },
};

export const LongName: Story = {
  args: {
    product: {
      ...baseProduct,
      name: "Premium Noise-Cancelling Over-Ear Wireless Bluetooth Headphones with 40-Hour Battery Life",
    },
  },
};

export const DraftProduct: Story = {
  args: {
    product: { ...baseProduct, status: "draft" as const },
  },
};
