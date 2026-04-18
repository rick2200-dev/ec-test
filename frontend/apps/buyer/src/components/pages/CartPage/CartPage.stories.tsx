import type { Meta, StoryObj } from "@storybook/react";
import { CartPagePresenter } from "./CartPage.presenter";

const meta: Meta<typeof CartPagePresenter> = {
  component: CartPagePresenter,
  title: "Pages/CartPage",
};
export default meta;

type Story = StoryObj<typeof CartPagePresenter>;

const labels = {
  title: "カート",
  emptyTitle: "カートは空です",
  emptyDescription: "商品を追加してください",
  browseCta: "商品を探す",
  itemColumnLabel: "商品",
  qtyColumnLabel: "数量",
  unitPriceColumnLabel: "単価",
  lineTotalColumnLabel: "小計",
  totalLabel: "合計",
  checkoutCta: "チェックアウトへ進む",
  loadingLabel: "読み込み中...",
  loginRequiredTitle: "カートを表示するにはログインしてください",
  loginRequiredCta: "ログインする",
} as const;

export const Loading: Story = {
  args: { ...labels, items: null, needsLogin: false },
};

export const Empty: Story = {
  args: { ...labels, items: [], needsLogin: false },
};

export const NeedsLogin: Story = {
  args: { ...labels, items: [], needsLogin: true },
};

export const ThreeItems: Story = {
  args: {
    ...labels,
    needsLogin: false,
    items: [
      {
        sku_id: "11111111-1111-1111-1111-111111111111",
        seller_id: "22222222-2222-2222-2222-222222222222",
        quantity: 2,
        unit_price_snapshot: 1500,
        currency: "JPY",
        product_name_snapshot: "オーガニックコットンTシャツ",
        sku_code_snapshot: "OCT-WHT-M",
        added_at: "2026-04-18T00:00:00Z",
      },
      {
        sku_id: "33333333-3333-3333-3333-333333333333",
        seller_id: "22222222-2222-2222-2222-222222222222",
        quantity: 1,
        unit_price_snapshot: 4980,
        currency: "JPY",
        product_name_snapshot: "デニムパンツ",
        sku_code_snapshot: "DNM-IDG-32",
        added_at: "2026-04-18T00:00:00Z",
      },
    ],
  },
};
