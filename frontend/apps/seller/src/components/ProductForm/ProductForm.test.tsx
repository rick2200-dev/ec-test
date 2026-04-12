import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import {
  ProductFormPresenter,
  type ProductFormPresenterProps,
} from "./ProductForm.presenter";

const categories = [
  { id: "cat-1", name: "ファッション", slug: "fashion", parentId: null },
  { id: "cat-2", name: "食品", slug: "food", parentId: null },
];

const baseProps: ProductFormPresenterProps = {
  name: "",
  slug: "",
  description: "",
  categoryId: "",
  status: "draft",
  categories,
  onNameChange: vi.fn(),
  onSlugChange: vi.fn(),
  onDescriptionChange: vi.fn(),
  onCategoryChange: vi.fn(),
  onStatusChange: vi.fn(),
};

describe("ProductFormPresenter", () => {
  it("renders form fields", () => {
    render(<ProductFormPresenter {...baseProps} />);
    expect(screen.getByLabelText(/商品名/)).toBeInTheDocument();
    expect(screen.getByLabelText(/スラッグ/)).toBeInTheDocument();
    expect(screen.getByLabelText(/説明/)).toBeInTheDocument();
    expect(screen.getByLabelText(/カテゴリ/)).toBeInTheDocument();
    expect(screen.getByLabelText(/ステータス/)).toBeInTheDocument();
  });

  it("renders category options", () => {
    render(<ProductFormPresenter {...baseProps} />);
    expect(screen.getByText("ファッション")).toBeInTheDocument();
    expect(screen.getByText("食品")).toBeInTheDocument();
    expect(screen.getByText("カテゴリを選択")).toBeInTheDocument();
  });

  it("renders status options", () => {
    render(<ProductFormPresenter {...baseProps} />);
    expect(screen.getByText("下書き")).toBeInTheDocument();
    expect(screen.getByText("公開")).toBeInTheDocument();
  });

  it("displays current values", () => {
    render(
      <ProductFormPresenter
        {...baseProps}
        name="テスト商品"
        slug="test-product"
        description="説明文"
        categoryId="cat-1"
        status="active"
      />
    );
    expect(screen.getByDisplayValue("テスト商品")).toBeInTheDocument();
    expect(screen.getByDisplayValue("test-product")).toBeInTheDocument();
    expect(screen.getByDisplayValue("説明文")).toBeInTheDocument();
  });

  it("calls onNameChange when name is typed", async () => {
    const onNameChange = vi.fn();
    render(<ProductFormPresenter {...baseProps} onNameChange={onNameChange} />);
    await userEvent.type(screen.getByLabelText(/商品名/), "a");
    expect(onNameChange).toHaveBeenCalled();
  });

  it("calls onSlugChange when slug is typed", async () => {
    const onSlugChange = vi.fn();
    render(<ProductFormPresenter {...baseProps} onSlugChange={onSlugChange} />);
    await userEvent.type(screen.getByLabelText(/スラッグ/), "x");
    expect(onSlugChange).toHaveBeenCalled();
  });

  it("calls onDescriptionChange when description is typed", async () => {
    const onDescriptionChange = vi.fn();
    render(
      <ProductFormPresenter
        {...baseProps}
        onDescriptionChange={onDescriptionChange}
      />
    );
    await userEvent.type(screen.getByLabelText(/説明/), "z");
    expect(onDescriptionChange).toHaveBeenCalled();
  });

  it("calls onCategoryChange when category is selected", async () => {
    const onCategoryChange = vi.fn();
    render(
      <ProductFormPresenter
        {...baseProps}
        onCategoryChange={onCategoryChange}
      />
    );
    await userEvent.selectOptions(screen.getByLabelText(/カテゴリ/), "cat-2");
    expect(onCategoryChange).toHaveBeenCalledWith("cat-2");
  });

  it("calls onStatusChange when status is selected", async () => {
    const onStatusChange = vi.fn();
    render(
      <ProductFormPresenter {...baseProps} onStatusChange={onStatusChange} />
    );
    await userEvent.selectOptions(screen.getByLabelText(/ステータス/), "active");
    expect(onStatusChange).toHaveBeenCalledWith("active");
  });
});
