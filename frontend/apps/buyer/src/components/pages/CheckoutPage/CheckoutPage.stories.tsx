import type { Meta, StoryObj } from "@storybook/react";
import { OrderSummaryPresenter } from "./OrderSummary.presenter";
import { CouponPreviewPanelPresenter } from "./CouponPreviewPanel.presenter";
import { PointsRedeemSliderPresenter } from "./PointsRedeemSlider.presenter";

const summaryLabels = {
  title: "ご注文内容",
  items: "商品数",
  subtotal: "小計",
  couponDiscount: "クーポン割引",
  pointsRedeemed: "ポイント利用",
  total: "合計",
};

const couponLabels = {
  title: "クーポン",
  placeholder: "クーポンコードを入力",
  apply: "適用",
  applying: "確認中...",
  clear: "解除",
  appliedDiscount: "割引額",
};

const pointsLabels = {
  title: "ポイントを利用",
  spendableLabel: "利用可能",
  useLabel: "利用",
  noPoints: "利用可能なポイントはありません",
};

const summaryMeta: Meta<typeof OrderSummaryPresenter> = {
  component: OrderSummaryPresenter,
  title: "Pages/CheckoutPage/OrderSummary",
};
export default summaryMeta;

type SummaryStory = StoryObj<typeof OrderSummaryPresenter>;

export const SubtotalOnly: SummaryStory = {
  args: {
    itemsCount: 2,
    subtotal: 5000,
    couponDiscount: 0,
    pointsRedeemed: 0,
    total: 5000,
    labels: summaryLabels,
  },
};

export const WithCoupon: SummaryStory = {
  args: {
    itemsCount: 2,
    subtotal: 5000,
    couponDiscount: 1000,
    pointsRedeemed: 0,
    total: 4000,
    labels: summaryLabels,
  },
};

export const WithPoints: SummaryStory = {
  args: {
    itemsCount: 3,
    subtotal: 5000,
    couponDiscount: 0,
    pointsRedeemed: 500,
    total: 4500,
    labels: summaryLabels,
  },
};

export const FullStack: SummaryStory = {
  args: {
    itemsCount: 3,
    subtotal: 5000,
    couponDiscount: 1000,
    pointsRedeemed: 500,
    total: 3500,
    labels: summaryLabels,
  },
};

// Coupon panel variants — exposed under their own title via meta override.
export const CouponEmpty: StoryObj<typeof CouponPreviewPanelPresenter> = {
  render: (args) => <CouponPreviewPanelPresenter {...args} />,
  args: {
    code: "",
    onCodeChange: () => {},
    onApply: () => {},
    onClear: () => {},
    loading: false,
    preview: null,
    labels: couponLabels,
  },
  parameters: { docs: { storyTitle: "Pages/CheckoutPage/CouponPanel/Empty" } },
};

export const CouponApplied: StoryObj<typeof CouponPreviewPanelPresenter> = {
  render: (args) => <CouponPreviewPanelPresenter {...args} />,
  args: {
    code: "SUMMER20",
    onCodeChange: () => {},
    onApply: () => {},
    onClear: () => {},
    loading: false,
    preview: {
      coupon_id: "c1",
      title: "Summer 20% off",
      description: "夏のセール限定",
      discount_amount: 1000,
    },
    labels: couponLabels,
  },
};

export const CouponError: StoryObj<typeof CouponPreviewPanelPresenter> = {
  render: (args) => <CouponPreviewPanelPresenter {...args} />,
  args: {
    code: "BADCODE",
    onCodeChange: () => {},
    onApply: () => {},
    onClear: () => {},
    loading: false,
    preview: null,
    error: "クーポンが見つかりません",
    labels: couponLabels,
  },
};

// Points slider variants.
export const PointsEmpty: StoryObj<typeof PointsRedeemSliderPresenter> = {
  render: (args) => <PointsRedeemSliderPresenter {...args} />,
  args: {
    spendable: 0,
    maxRedeemable: 1000,
    value: 0,
    onChange: () => {},
    labels: pointsLabels,
  },
};

export const PointsAvailable: StoryObj<typeof PointsRedeemSliderPresenter> = {
  render: (args) => <PointsRedeemSliderPresenter {...args} />,
  args: {
    spendable: 1500,
    maxRedeemable: 1000,
    value: 250,
    onChange: () => {},
    labels: pointsLabels,
  },
};
