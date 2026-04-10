import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { ProductCardPresenter } from "./ProductCard.presenter";

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...props
  }: {
    children: React.ReactNode;
    href: string;
    [key: string]: unknown;
  }) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));

describe("ProductCardPresenter", () => {
  const baseProps = {
    href: "/products/test-product",
    name: "Test Product",
    sellerName: "Test Seller",
    priceLabel: "¥12,800",
  };

  it("renders product name", () => {
    render(<ProductCardPresenter {...baseProps} />);
    expect(screen.getByText("Test Product")).toBeInTheDocument();
  });

  it("renders formatted price", () => {
    render(<ProductCardPresenter {...baseProps} />);
    expect(screen.getByText("¥12,800")).toBeInTheDocument();
  });

  it("links to product detail page", () => {
    render(<ProductCardPresenter {...baseProps} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/products/test-product");
  });

  it("renders seller name", () => {
    render(<ProductCardPresenter {...baseProps} />);
    expect(screen.getByText("Test Seller")).toBeInTheDocument();
  });

  it("omits seller line when not provided", () => {
    render(<ProductCardPresenter {...baseProps} sellerName={undefined} />);
    expect(screen.queryByText("Test Seller")).not.toBeInTheDocument();
  });
});
