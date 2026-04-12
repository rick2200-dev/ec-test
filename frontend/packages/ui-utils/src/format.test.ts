import { describe, it, expect } from "vitest";
import { formatCurrency } from "./format";

describe("formatCurrency", () => {
  it("formats zero", () => {
    expect(formatCurrency(0)).toBe("¥0");
  });

  it("formats small amount without separator", () => {
    expect(formatCurrency(999)).toBe("¥999");
  });

  it("formats amount with thousands separator", () => {
    const result = formatCurrency(12800);
    expect(result).toBe("¥12,800");
  });

  it("formats large amount", () => {
    const result = formatCurrency(1000000);
    expect(result).toBe("¥1,000,000");
  });

  it("formats negative amount", () => {
    const result = formatCurrency(-500);
    expect(result).toBe("¥-500");
  });
});
