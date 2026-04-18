import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AddToCartButtonPresenter } from "./AddToCartButton.presenter";

const baseProps = {
  label: "カートに追加",
  loadingLabel: "追加中...",
  successLabel: "追加しました",
  loginCta: "ログインして続ける",
  disabled: false,
  loading: false,
  succeeded: false,
  needsLogin: false,
  loginHref: "/auth/login?returnTo=%2F",
  onClick: () => {},
};

describe("AddToCartButtonPresenter", () => {
  it("renders the default label and triggers onClick", async () => {
    const onClick = vi.fn();
    render(<AddToCartButtonPresenter {...baseProps} onClick={onClick} />);
    await userEvent.click(screen.getByRole("button", { name: "カートに追加" }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("disables the button while loading", () => {
    render(<AddToCartButtonPresenter {...baseProps} loading />);
    expect(screen.getByRole("button", { name: "追加中..." })).toBeDisabled();
  });

  it("flips the label after success", () => {
    render(<AddToCartButtonPresenter {...baseProps} succeeded />);
    expect(screen.getByRole("button", { name: "追加しました" })).toBeInTheDocument();
  });

  it("renders a login link instead of the button when needsLogin is true", () => {
    render(<AddToCartButtonPresenter {...baseProps} needsLogin />);
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "ログインして続ける" })).toHaveAttribute(
      "href",
      "/auth/login?returnTo=%2F"
    );
  });

  it("shows an error message under the button", () => {
    render(<AddToCartButtonPresenter {...baseProps} errorMessage="在庫切れです" />);
    expect(screen.getByRole("alert")).toHaveTextContent("在庫切れです");
  });

  it("disables the button when no SKU is selected", () => {
    render(<AddToCartButtonPresenter {...baseProps} disabled />);
    expect(screen.getByRole("button")).toBeDisabled();
  });
});
