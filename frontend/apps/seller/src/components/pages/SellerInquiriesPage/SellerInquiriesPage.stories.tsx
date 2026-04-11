import type { Meta, StoryObj } from "@storybook/react";
import { SellerInquiriesPagePresenter } from "./SellerInquiriesPage.presenter";

const meta: Meta<typeof SellerInquiriesPagePresenter> = {
  component: SellerInquiriesPagePresenter,
  title: "Pages/SellerInquiriesPage",
};
export default meta;

type Story = StoryObj<typeof SellerInquiriesPagePresenter>;

export const Empty: Story = {
  args: {
    listSlot: (
      <div className="rounded-lg border border-dashed border-border bg-white py-12 text-center text-sm text-text-secondary">
        お問い合わせはまだありません
      </div>
    ),
  },
};
