import type { Meta, StoryObj } from "@storybook/react";
import { WriteReviewButtonPresenter } from "./WriteReviewButton.presenter";

const meta: Meta<typeof WriteReviewButtonPresenter> = {
  component: WriteReviewButtonPresenter,
  title: "Reviews/WriteReviewButton",
};
export default meta;

type Story = StoryObj<typeof WriteReviewButtonPresenter>;

const baseProps = {
  triggerLabel: "レビューを書く",
  modalTitle: "レビューを投稿",
  productName: "ワイヤレスイヤホン Pro",
  ratingLabel: "評価",
  ratingValue: 0,
  ratingHoverValue: null,
  onRatingChange: () => {},
  onRatingHover: () => {},
  titleLabel: "タイトル",
  titlePlaceholder: "例：音質が素晴らしい",
  titleValue: "",
  onTitleChange: () => {},
  bodyLabel: "レビュー内容",
  bodyPlaceholder: "商品の感想を書いてください",
  bodyValue: "",
  onBodyChange: () => {},
  onOpen: () => {},
  onClose: () => {},
  onSubmit: (e: React.FormEvent) => e.preventDefault(),
  submitLabel: "投稿する",
  submittingLabel: "送信中...",
  cancelLabel: "キャンセル",
};

export const Closed: Story = {
  args: { ...baseProps, open: false, submitting: false, error: null },
};

export const OpenEmpty: Story = {
  args: { ...baseProps, open: true, submitting: false, error: null },
};

export const OpenWithRating: Story = {
  args: {
    ...baseProps,
    open: true,
    submitting: false,
    ratingValue: 4,
    titleValue: "音質がとても良い",
    bodyValue: "ノイズキャンセリングも素晴らしく、とても満足しています。",
    error: null,
  },
};

export const OpenWithError: Story = {
  args: {
    ...baseProps,
    open: true,
    submitting: false,
    error: "この商品を購入していないため、レビューを投稿できません",
  },
};
