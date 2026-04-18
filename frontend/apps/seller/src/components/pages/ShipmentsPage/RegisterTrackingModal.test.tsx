import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { RegisterTrackingModalPresenter } from "./RegisterTrackingModal.presenter";

const labels = {
  title: "出荷登録",
  carrier: "配送業者",
  trackingNumber: "追跡番号",
  note: "メモ",
  submit: "登録",
  submitting: "登録中...",
  cancel: "キャンセル",
};

const baseProps = {
  open: true,
  carrier: "",
  trackingNumber: "",
  note: "",
  submitting: false,
  onCarrierChange: () => {},
  onTrackingNumberChange: () => {},
  onNoteChange: () => {},
  onSubmit: () => {},
  onClose: () => {},
  labels,
};

describe("RegisterTrackingModalPresenter", () => {
  it("renders nothing when closed", () => {
    const { container } = render(<RegisterTrackingModalPresenter {...baseProps} open={false} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("submits the form via the submit button", async () => {
    const onSubmit = vi.fn();
    render(
      <RegisterTrackingModalPresenter
        {...baseProps}
        carrier="ヤマト"
        trackingNumber="123"
        onSubmit={onSubmit}
      />
    );
    await userEvent.click(screen.getByRole("button", { name: "登録" }));
    expect(onSubmit).toHaveBeenCalled();
  });

  it("shows an inline error", () => {
    render(<RegisterTrackingModalPresenter {...baseProps} errorMessage="登録に失敗しました" />);
    expect(screen.getByRole("alert")).toHaveTextContent("登録に失敗しました");
  });

  it("disables buttons while submitting", () => {
    render(<RegisterTrackingModalPresenter {...baseProps} submitting />);
    expect(screen.getByRole("button", { name: "登録中..." })).toBeDisabled();
    expect(screen.getByRole("button", { name: "キャンセル" })).toBeDisabled();
  });
});
