import type { Meta, StoryObj } from "@storybook/react";
import { InquiriesPagePresenter } from "./InquiriesPage.presenter";

const meta: Meta<typeof InquiriesPagePresenter> = {
  component: InquiriesPagePresenter,
  title: "Pages/InquiriesPage",
};
export default meta;

type Story = StoryObj<typeof InquiriesPagePresenter>;

export const Empty: Story = {
  args: {
    listSlot: (
      <div className="rounded-lg border border-dashed border-gray-300 bg-white py-12 text-center text-sm text-gray-500">
        お問い合わせはまだありません
      </div>
    ),
  },
};

export const WithError: Story = {
  args: {
    errorSlot: (
      <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
        通信エラーが発生しました
      </div>
    ),
    listSlot: null,
  },
};
