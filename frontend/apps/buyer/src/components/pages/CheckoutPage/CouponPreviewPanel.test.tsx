import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CouponPreviewPanelPresenter } from "./CouponPreviewPanel.presenter";

const labels = {
  title: "クーポン",
  placeholder: "クーポンコード",
  apply: "適用",
  applying: "確認中...",
  clear: "解除",
  appliedDiscount: "割引額",
};

describe("CouponPreviewPanelPresenter", () => {
  it("disables apply when the code is empty", () => {
    render(
      <CouponPreviewPanelPresenter
        code=""
        onCodeChange={() => {}}
        onApply={() => {}}
        onClear={() => {}}
        loading={false}
        preview={null}
        labels={labels}
      />
    );
    expect(screen.getByRole("button", { name: "適用" })).toBeDisabled();
  });

  it("shows applied state with discount and clear button", () => {
    render(
      <CouponPreviewPanelPresenter
        code="SUMMER20"
        onCodeChange={() => {}}
        onApply={() => {}}
        onClear={() => {}}
        loading={false}
        preview={{
          coupon_id: "c1",
          title: "Summer 20% off",
          description: "夏のセール",
          discount_amount: 500,
        }}
        labels={labels}
      />
    );
    expect(screen.getByText("Summer 20% off")).toBeInTheDocument();
    expect(screen.getByText("−¥500")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "解除" })).toBeInTheDocument();
  });

  it("emits onApply when the user clicks apply", async () => {
    const onApply = vi.fn();
    render(
      <CouponPreviewPanelPresenter
        code="SUMMER20"
        onCodeChange={() => {}}
        onApply={onApply}
        onClear={() => {}}
        loading={false}
        preview={null}
        labels={labels}
      />
    );
    await userEvent.click(screen.getByRole("button", { name: "適用" }));
    expect(onApply).toHaveBeenCalledTimes(1);
  });

  it("renders an error message", () => {
    render(
      <CouponPreviewPanelPresenter
        code="BAD"
        onCodeChange={() => {}}
        onApply={() => {}}
        onClear={() => {}}
        loading={false}
        preview={null}
        error="クーポンが見つかりません"
        labels={labels}
      />
    );
    expect(screen.getByRole("alert")).toHaveTextContent("クーポンが見つかりません");
  });
});
