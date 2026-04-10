import type { Meta, StoryObj } from "@storybook/react";
import { useState } from "react";
import { SKUManagerPresenter, type SKUInput } from "./SKUManager.presenter";

const meta: Meta<typeof SKUManagerPresenter> = {
  title: "Seller/SKUManager",
  component: SKUManagerPresenter,
  parameters: { layout: "padded" },
};

export default meta;
type Story = StoryObj<typeof SKUManagerPresenter>;

const noop = () => {};

export const SingleSKU: Story = {
  args: {
    skus: [{ code: "", price: "", color: "", size: "" }],
    onAdd: noop,
    onRemove: noop,
    onUpdate: noop,
  },
};

export const MultipleSKUs: Story = {
  args: {
    skus: [
      { code: "OCT-WHT-M", price: "3980", color: "ホワイト", size: "M" },
      { code: "OCT-BLK-M", price: "3980", color: "ブラック", size: "M" },
      { code: "OCT-WHT-L", price: "3980", color: "ホワイト", size: "L" },
    ],
    onAdd: noop,
    onRemove: noop,
    onUpdate: noop,
  },
};

export const Interactive: Story = {
  render: () => {
    const [skus, setSkus] = useState<SKUInput[]>([
      { code: "", price: "", color: "", size: "" },
    ]);
    return (
      <SKUManagerPresenter
        skus={skus}
        onAdd={() => setSkus([...skus, { code: "", price: "", color: "", size: "" }])}
        onRemove={(i) => setSkus(skus.filter((_, idx) => idx !== i))}
        onUpdate={(i, field, value) => {
          const next = [...skus];
          next[i] = { ...next[i], [field]: value };
          setSkus(next);
        }}
      />
    );
  },
};
