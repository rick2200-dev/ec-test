import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import {
  CancelOrderButtonPresenter,
  type CancelOrderButtonPresenterProps,
} from "./CancelOrderButton.presenter";

const baseProps: CancelOrderButtonPresenterProps = {
  triggerLabel: "注文をキャンセル",
  modalTitle: "キャンセルリクエスト",
  modalDescription: "キャンセル理由を入力してください",
  reasonLabel: "理由",
  reasonPlaceholder: "理由を入力...",
  reasonValue: "",
  onReasonChange: vi.fn(),
  open: false,
  onOpen: vi.fn(),
  onClose: vi.fn(),
  onSubmit: vi.fn(),
  submitLabel: "送信",
  submittingLabel: "送信中...",
  cancelLabel: "戻る",
  submitting: false,
  error: null,
};

describe("CancelOrderButtonPresenter", () => {
  it("renders trigger button", () => {
    render(<CancelOrderButtonPresenter {...baseProps} />);
    expect(screen.getByText("注文をキャンセル")).toBeInTheDocument();
  });

  it("does not render modal when closed", () => {
    render(<CancelOrderButtonPresenter {...baseProps} open={false} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders modal when open", () => {
    render(<CancelOrderButtonPresenter {...baseProps} open={true} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("キャンセルリクエスト")).toBeInTheDocument();
  });

  it("calls onOpen when trigger button is clicked", async () => {
    const onOpen = vi.fn();
    render(<CancelOrderButtonPresenter {...baseProps} onOpen={onOpen} />);
    await userEvent.click(screen.getByText("注文をキャンセル"));
    expect(onOpen).toHaveBeenCalledOnce();
  });

  it("calls onClose when cancel button is clicked", async () => {
    const onClose = vi.fn();
    render(
      <CancelOrderButtonPresenter {...baseProps} open={true} onClose={onClose} />
    );
    await userEvent.click(screen.getByText("戻る"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onReasonChange when textarea is typed into", async () => {
    const onReasonChange = vi.fn();
    render(
      <CancelOrderButtonPresenter
        {...baseProps}
        open={true}
        onReasonChange={onReasonChange}
      />
    );
    const textarea = screen.getByPlaceholderText("理由を入力...");
    await userEvent.type(textarea, "a");
    expect(onReasonChange).toHaveBeenCalled();
  });

  it("shows submit label when not submitting", () => {
    render(<CancelOrderButtonPresenter {...baseProps} open={true} />);
    expect(screen.getByText("送信")).toBeInTheDocument();
  });

  it("shows submitting label and disables button when submitting", () => {
    render(
      <CancelOrderButtonPresenter {...baseProps} open={true} submitting={true} />
    );
    const submitBtn = screen.getByText("送信中...");
    expect(submitBtn).toBeInTheDocument();
    expect(submitBtn).toBeDisabled();
  });

  it("displays error message when error is set", () => {
    render(
      <CancelOrderButtonPresenter
        {...baseProps}
        open={true}
        error="キャンセルできません"
      />
    );
    expect(screen.getByRole("alert")).toHaveTextContent("キャンセルできません");
  });

  it("does not display error when error is null", () => {
    render(<CancelOrderButtonPresenter {...baseProps} open={true} error={null} />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
