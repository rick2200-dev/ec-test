import type { Meta, StoryObj } from "@storybook/react";
import { SellerInquiryDetailPagePresenter } from "./SellerInquiryDetailPage.presenter";

const meta: Meta<typeof SellerInquiryDetailPagePresenter> = {
  component: SellerInquiryDetailPagePresenter,
  title: "Pages/SellerInquiryDetailPage",
};
export default meta;

type Story = StoryObj<typeof SellerInquiryDetailPagePresenter>;

export const Loading: Story = {
  args: {
    threadSlot: <div className="text-sm text-text-secondary">...</div>,
  },
};
