import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import {
  SKUManagerPresenter,
  type SKUInput,
} from "./SKUManager.presenter";

const emptySKU: SKUInput = { code: "", price: "", color: "", size: "" };

describe("SKUManagerPresenter", () => {
  it("renders section heading", () => {
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    expect(screen.getByText("SKU（バリエーション）")).toBeInTheDocument();
  });

  it("renders add button", () => {
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    expect(screen.getByText("SKUを追加")).toBeInTheDocument();
  });

  it("renders one SKU row", () => {
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    expect(screen.getByText("SKU #1")).toBeInTheDocument();
  });

  it("renders multiple SKU rows", () => {
    render(
      <SKUManagerPresenter
        skus={[emptySKU, emptySKU, emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    expect(screen.getByText("SKU #1")).toBeInTheDocument();
    expect(screen.getByText("SKU #2")).toBeInTheDocument();
    expect(screen.getByText("SKU #3")).toBeInTheDocument();
  });

  it("does not show remove button for single SKU", () => {
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    expect(screen.queryByText("削除")).not.toBeInTheDocument();
  });

  it("shows remove button when multiple SKUs exist", () => {
    render(
      <SKUManagerPresenter
        skus={[emptySKU, emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    const removeButtons = screen.getAllByText("削除");
    expect(removeButtons).toHaveLength(2);
  });

  it("calls onAdd when add button is clicked", async () => {
    const onAdd = vi.fn();
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={onAdd}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    await userEvent.click(screen.getByText("SKUを追加"));
    expect(onAdd).toHaveBeenCalledOnce();
  });

  it("calls onRemove with correct index", async () => {
    const onRemove = vi.fn();
    render(
      <SKUManagerPresenter
        skus={[emptySKU, emptySKU]}
        onAdd={vi.fn()}
        onRemove={onRemove}
        onUpdate={vi.fn()}
      />
    );
    const removeButtons = screen.getAllByText("削除");
    await userEvent.click(removeButtons[1]);
    expect(onRemove).toHaveBeenCalledWith(1);
  });

  it("calls onUpdate when code input is changed", async () => {
    const onUpdate = vi.fn();
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={onUpdate}
      />
    );
    await userEvent.type(screen.getByLabelText(/SKUコード/), "A");
    expect(onUpdate).toHaveBeenCalledWith(0, "code", "A");
  });

  it("calls onUpdate when price input is changed", async () => {
    const onUpdate = vi.fn();
    render(
      <SKUManagerPresenter
        skus={[emptySKU]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={onUpdate}
      />
    );
    await userEvent.type(screen.getByLabelText(/価格/), "1");
    expect(onUpdate).toHaveBeenCalledWith(0, "price", "1");
  });

  it("displays existing SKU values", () => {
    const sku: SKUInput = {
      code: "OCT-WHT-M",
      price: "3980",
      color: "ホワイト",
      size: "M",
    };
    render(
      <SKUManagerPresenter
        skus={[sku]}
        onAdd={vi.fn()}
        onRemove={vi.fn()}
        onUpdate={vi.fn()}
      />
    );
    expect(screen.getByDisplayValue("OCT-WHT-M")).toBeInTheDocument();
    expect(screen.getByDisplayValue("3980")).toBeInTheDocument();
    expect(screen.getByDisplayValue("ホワイト")).toBeInTheDocument();
    expect(screen.getByDisplayValue("M")).toBeInTheDocument();
  });
});
