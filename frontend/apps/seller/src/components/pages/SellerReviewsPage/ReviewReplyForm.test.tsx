import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ReviewReplyFormPresenter } from "./ReviewReplyForm.presenter";

const labels = {
  createTitle: "レビューに返信",
  editTitle: "返信を編集",
  bodyLabel: "返信内容",
  bodyPlaceholder: "返信を入力",
  submit: "送信",
  submitting: "送信中...",
  cancel: "キャンセル",
  delete: "削除",
  deleting: "削除中...",
};

const baseProps = {
  open: true,
  body: "",
  onBodyChange: () => {},
  submitting: false,
  deleting: false,
  onSubmit: () => {},
  onClose: () => {},
  labels,
};

describe("ReviewReplyFormPresenter", () => {
  it("renders nothing when closed", () => {
    const { container } = render(
      <ReviewReplyFormPresenter {...baseProps} open={false} mode="create" />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("shows the create title and disables submit when body is empty", () => {
    render(<ReviewReplyFormPresenter {...baseProps} mode="create" body="" />);
    expect(screen.getByText("レビューに返信")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "送信" })).toBeDisabled();
  });

  it("shows the edit title plus a delete button in edit mode", () => {
    const onDelete = vi.fn();
    render(
      <ReviewReplyFormPresenter {...baseProps} mode="edit" body="既存の返信" onDelete={onDelete} />
    );
    expect(screen.getByText("返信を編集")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "削除" })).toBeInTheDocument();
  });

  it("calls onSubmit when the form submits with a non-empty body", async () => {
    const onSubmit = vi.fn();
    render(
      <ReviewReplyFormPresenter
        {...baseProps}
        mode="create"
        body="ありがとうございます。"
        onSubmit={onSubmit}
      />
    );
    await userEvent.click(screen.getByRole("button", { name: "送信" }));
    expect(onSubmit).toHaveBeenCalled();
  });
});
