import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { StatusBadgePresenter } from "./StatusBadge.presenter";

describe("StatusBadgePresenter", () => {
  it("renders the provided label", () => {
    render(<StatusBadgePresenter tone="success" label="有効" />);
    expect(screen.getByText("有効")).toBeInTheDocument();
  });

  it("applies green styling for success tone", () => {
    const { container } = render(<StatusBadgePresenter tone="success" label="ok" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-green-100");
    expect(badge.className).toContain("text-green-800");
  });

  it("applies yellow styling for warning tone", () => {
    const { container } = render(<StatusBadgePresenter tone="warning" label="warn" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-yellow-100");
  });

  it("applies red styling for danger tone", () => {
    const { container } = render(<StatusBadgePresenter tone="danger" label="bad" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-red-100");
  });

  it("applies gray styling for neutral tone", () => {
    const { container } = render(<StatusBadgePresenter tone="neutral" label="?" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-gray-100");
  });
});
