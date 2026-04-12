import { describe, it, expect } from "vitest";
import { STATUS_LABELS, STATUS_COLORS } from "./status";

describe("STATUS_LABELS", () => {
  it("has label for all order statuses", () => {
    const orderStatuses = [
      "pending",
      "processing",
      "shipped",
      "delivered",
      "completed",
      "cancelled",
    ];
    for (const status of orderStatuses) {
      expect(STATUS_LABELS[status]).toBeDefined();
      expect(typeof STATUS_LABELS[status]).toBe("string");
      expect(STATUS_LABELS[status].length).toBeGreaterThan(0);
    }
  });

  it("has label for all product statuses", () => {
    const productStatuses = ["draft", "active", "archived"];
    for (const status of productStatuses) {
      expect(STATUS_LABELS[status]).toBeDefined();
    }
  });

  it("returns Japanese labels", () => {
    expect(STATUS_LABELS["pending"]).toBe("未処理");
    expect(STATUS_LABELS["cancelled"]).toBe("キャンセル");
    expect(STATUS_LABELS["draft"]).toBe("下書き");
    expect(STATUS_LABELS["active"]).toBe("公開中");
  });

  it("returns undefined for unknown status", () => {
    expect(STATUS_LABELS["nonexistent"]).toBeUndefined();
  });
});

describe("STATUS_COLORS", () => {
  it("has color for every status that has a label", () => {
    for (const key of Object.keys(STATUS_LABELS)) {
      expect(STATUS_COLORS[key]).toBeDefined();
    }
  });

  it("returns Tailwind class strings", () => {
    for (const value of Object.values(STATUS_COLORS)) {
      expect(value).toMatch(/^bg-\w+-\d+\s+text-\w+-\d+$/);
    }
  });

  it("has correct color for specific statuses", () => {
    expect(STATUS_COLORS["completed"]).toContain("bg-green");
    expect(STATUS_COLORS["cancelled"]).toContain("bg-red");
    expect(STATUS_COLORS["pending"]).toContain("bg-yellow");
  });

  it("returns undefined for unknown status", () => {
    expect(STATUS_COLORS["nonexistent"]).toBeUndefined();
  });
});
