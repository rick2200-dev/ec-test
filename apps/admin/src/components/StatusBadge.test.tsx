import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { NextIntlClientProvider } from "next-intl";
import StatusBadge from "./StatusBadge";

const messages = {
  status: {
    active: "有効",
    approved: "承認済み",
    pending: "承認待ち",
    suspended: "停止中",
    operational: "正常",
    degraded: "低下",
    down: "停止",
  },
};

function renderWithIntl(ui: React.ReactElement) {
  return render(
    <NextIntlClientProvider locale="ja" messages={messages}>
      {ui}
    </NextIntlClientProvider>
  );
}

describe("StatusBadge", () => {
  it("renders active status label", () => {
    renderWithIntl(<StatusBadge status="active" />);
    expect(screen.getByText("有効")).toBeInTheDocument();
  });

  it("renders pending status label", () => {
    renderWithIntl(<StatusBadge status="pending" />);
    expect(screen.getByText("承認待ち")).toBeInTheDocument();
  });

  it("renders suspended status label", () => {
    renderWithIntl(<StatusBadge status="suspended" />);
    expect(screen.getByText("停止中")).toBeInTheDocument();
  });

  it("applies green styling for active status", () => {
    const { container } = renderWithIntl(<StatusBadge status="active" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-green-100");
    expect(badge.className).toContain("text-green-800");
  });

  it("applies yellow styling for pending status", () => {
    const { container } = renderWithIntl(<StatusBadge status="pending" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-yellow-100");
  });

  it("applies red styling for down status", () => {
    const { container } = renderWithIntl(<StatusBadge status="down" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain("bg-red-100");
  });
});
