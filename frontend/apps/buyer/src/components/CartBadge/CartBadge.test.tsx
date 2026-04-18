import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CartBadgePresenter } from "./CartBadge.presenter";

describe("CartBadgePresenter", () => {
  it("hides the badge when count is 0", () => {
    render(<CartBadgePresenter count={0} ariaLabel="Cart" />);
    expect(screen.getByLabelText("Cart")).toBeInTheDocument();
    expect(screen.queryByTestId("cart-badge-count")).not.toBeInTheDocument();
  });

  it("shows the count when non-zero", () => {
    render(<CartBadgePresenter count={3} ariaLabel="Cart" />);
    expect(screen.getByTestId("cart-badge-count")).toHaveTextContent("3");
  });

  it("caps the displayed count at 99+", () => {
    render(<CartBadgePresenter count={150} ariaLabel="Cart" />);
    expect(screen.getByTestId("cart-badge-count")).toHaveTextContent("99+");
  });
});
