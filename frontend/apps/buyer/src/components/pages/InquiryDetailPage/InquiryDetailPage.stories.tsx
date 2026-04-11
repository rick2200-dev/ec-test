import type { Meta, StoryObj } from "@storybook/react";
import { InquiryDetailPagePresenter } from "./InquiryDetailPage.presenter";

const meta: Meta<typeof InquiryDetailPagePresenter> = {
  component: InquiryDetailPagePresenter,
  title: "Pages/InquiryDetailPage",
};
export default meta;

type Story = StoryObj<typeof InquiryDetailPagePresenter>;

export const Loading: Story = {
  args: {
    threadSlot: <div className="text-sm text-gray-500">...</div>,
  },
};

export const ErrorState: Story = {
  args: {
    threadSlot: (
      <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
        通信エラーが発生しました
      </div>
    ),
  },
};
