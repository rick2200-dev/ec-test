import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PointsRedeemSliderPresenter } from "./PointsRedeemSlider.presenter";

const labels = {
  title: "ポイント利用",
  spendableLabel: "利用可能",
  useLabel: "利用",
  noPoints: "利用可能なポイントはありません",
};

describe("PointsRedeemSliderPresenter", () => {
  it("shows the no-points message when balance is 0", () => {
    render(
      <PointsRedeemSliderPresenter
        spendable={0}
        maxRedeemable={1000}
        value={0}
        onChange={() => {}}
        labels={labels}
      />
    );
    expect(screen.getByText("利用可能なポイントはありません")).toBeInTheDocument();
    expect(screen.queryByRole("slider")).not.toBeInTheDocument();
  });

  it("renders a slider capped at min(spendable, maxRedeemable)", () => {
    render(
      <PointsRedeemSliderPresenter
        spendable={500}
        maxRedeemable={300}
        value={100}
        onChange={() => {}}
        labels={labels}
      />
    );
    const slider = screen.getByRole("slider");
    expect(slider).toHaveAttribute("max", "300");
    expect(slider).toHaveAttribute("value", "100");
  });

  it("emits onChange when the slider is moved", () => {
    const onChange = vi.fn();
    render(
      <PointsRedeemSliderPresenter
        spendable={500}
        maxRedeemable={500}
        value={0}
        onChange={onChange}
        labels={labels}
      />
    );
    const slider = screen.getByRole("slider") as HTMLInputElement;
    // jsdom doesn't translate keyboard events on range inputs into change
    // events; dispatch the change directly.
    fireEvent.change(slider, { target: { value: "200" } });
    expect(onChange).toHaveBeenCalledWith(200);
  });
});
