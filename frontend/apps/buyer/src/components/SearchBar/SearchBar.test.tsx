import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { SearchBarPresenter } from "./SearchBar.presenter";

const baseProps = {
  value: "",
  onChange: () => {},
  onSubmit: () => {},
  onPickSuggestion: () => {},
  suggestions: [],
  loading: false,
  placeholder: "商品を検索...",
  searchAriaLabel: "検索",
};

describe("SearchBarPresenter", () => {
  it("renders the input and submit button", () => {
    render(<SearchBarPresenter {...baseProps} />);
    expect(screen.getByPlaceholderText("商品を検索...")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "検索" })).toBeInTheDocument();
  });

  it("calls onSubmit when the form submits", async () => {
    const onSubmit = vi.fn();
    render(<SearchBarPresenter {...baseProps} onSubmit={onSubmit} />);
    await userEvent.click(screen.getByRole("button", { name: "検索" }));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("shows suggestions and triggers onPickSuggestion", async () => {
    const onPickSuggestion = vi.fn();
    render(
      <SearchBarPresenter
        {...baseProps}
        suggestions={["headphones", "headset", "headlamp"]}
        onPickSuggestion={onPickSuggestion}
      />
    );
    await userEvent.click(screen.getByRole("button", { name: "headset" }));
    expect(onPickSuggestion).toHaveBeenCalledWith("headset");
  });

  it("renders the loading placeholder while suggestions resolve", () => {
    render(<SearchBarPresenter {...baseProps} loading suggestions={[]} />);
    expect(screen.getByText("…")).toBeInTheDocument();
  });
});
