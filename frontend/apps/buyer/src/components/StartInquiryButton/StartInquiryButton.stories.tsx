import type { Meta, StoryObj } from "@storybook/react";
import { StartInquiryButtonPresenter } from "./StartInquiryButton.presenter";

const meta: Meta<typeof StartInquiryButtonPresenter> = {
  component: StartInquiryButtonPresenter,
  title: "Inquiries/StartInquiryButton",
};
export default meta;

type Story = StoryObj<typeof StartInquiryButtonPresenter>;

const baseProps = {
  triggerLabel: "出品者に問い合わせる",
  modalTitle: "出品者に問い合わせる",
  productName: "ワイヤレスイヤホン Pro",
  skuCode: "SKU-EP-001",
  subjectLabel: "件名",
  subjectPlaceholder: "例：発送時期について",
  subjectValue: "",
  onSubjectChange: () => {},
  bodyLabel: "最初のメッセージ",
  bodyPlaceholder: "出品者に伝えたい内容を入力してください",
  bodyValue: "",
  onBodyChange: () => {},
  onOpen: () => {},
  onClose: () => {},
  onSubmit: (e: React.FormEvent) => e.preventDefault(),
  submitLabel: "送信する",
  submittingLabel: "送信中...",
  cancelLabel: "キャンセル",
};

export const Closed: Story = {
  args: { ...baseProps, open: false, submitting: false, error: null },
};

export const OpenEmpty: Story = {
  args: { ...baseProps, open: true, submitting: false, error: null },
};

export const OpenWithError: Story = {
  args: {
    ...baseProps,
    open: true,
    submitting: false,
    subjectValue: "発送時期について",
    bodyValue: "いつ発送されますか？",
    error: "この商品には問い合わせできません",
  },
};
