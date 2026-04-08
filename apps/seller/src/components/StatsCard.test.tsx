import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import StatsCard from "./StatsCard";

describe("StatsCard", () => {
  it("renders title and value", () => {
    render(<StatsCard title="今日の売上" value="¥28,740" />);
    expect(screen.getByText("今日の売上")).toBeInTheDocument();
    expect(screen.getByText("¥28,740")).toBeInTheDocument();
  });

  it("renders subtitle when provided", () => {
    render(<StatsCard title="売上" value="¥100" subtitle="前日比 +12.5%" />);
    expect(screen.getByText("前日比 +12.5%")).toBeInTheDocument();
  });

  it("does not render subtitle when not provided", () => {
    const { container } = render(<StatsCard title="売上" value="¥100" />);
    const paragraphs = container.querySelectorAll("p");
    expect(paragraphs).toHaveLength(2);
  });

  it("applies accent color class", () => {
    const { container } = render(<StatsCard title="Alert" value="5" accent="danger" />);
    const card = container.firstChild as HTMLElement;
    expect(card.className).toContain("border-l-danger");
  });

  it("defaults to default accent", () => {
    const { container } = render(<StatsCard title="Stat" value="10" />);
    const card = container.firstChild as HTMLElement;
    expect(card.className).toContain("border-l-accent");
  });
});
