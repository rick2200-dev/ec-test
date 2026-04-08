import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import ProductCard from "./ProductCard";

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

vi.mock("@/lib/mock-data", () => ({
  formatPrice: (amount: number) => `¥${amount.toLocaleString()}`,
  getLowestPrice: () => 12800,
  getSellerById: () => ({ id: "s1", name: "Test Seller" }),
}));

const product = {
  id: "1",
  tenant_id: "t1",
  seller_id: "s1",
  name: "Test Product",
  slug: "test-product",
  description: "A test product",
  status: "active" as const,
  created_at: "2025-01-01",
  updated_at: "2025-01-01",
};

describe("ProductCard", () => {
  it("renders product name", () => {
    render(<ProductCard product={product} />);
    expect(screen.getByText("Test Product")).toBeInTheDocument();
  });

  it("renders formatted price", () => {
    render(<ProductCard product={product} />);
    expect(screen.getByText("¥12,800")).toBeInTheDocument();
  });

  it("links to product detail page", () => {
    render(<ProductCard product={product} />);
    const link = screen.getByRole("link");
    expect(link).toHaveAttribute("href", "/products/test-product");
  });

  it("renders seller name", () => {
    render(<ProductCard product={product} />);
    expect(screen.getByText("Test Seller")).toBeInTheDocument();
  });
});
