import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ShipmentStatus } from "@ec-marketplace/types";
import { ShipmentTrackingBlockPresenter } from "./ShipmentTrackingBlock.presenter";

const statusLabels: Record<ShipmentStatus, string> = {
  pending: "準備中",
  ready_to_ship: "出荷準備完了",
  shipped: "発送済み",
  delivered: "配達完了",
  cancelled: "キャンセル",
};

const baseLabels = {
  sectionTitle: "配送状況",
  preparing: "出荷準備中",
  carrier: "配送業者",
  trackingNumber: "追跡番号",
  shippedAt: "発送日時",
  deliveredAt: "配達日時",
  statusLabels,
};

describe("ShipmentTrackingBlockPresenter", () => {
  it("renders the preparing state when there is no shipment", () => {
    render(<ShipmentTrackingBlockPresenter status={null} labels={baseLabels} locale="ja" />);
    expect(screen.getByText("出荷準備中")).toBeInTheDocument();
  });

  it("renders carrier/tracking when shipped", () => {
    render(
      <ShipmentTrackingBlockPresenter
        status="shipped"
        carrier="ヤマト運輸"
        trackingNumber="1234-5678"
        shippedAt="2026-04-01T10:00:00Z"
        labels={baseLabels}
        locale="ja"
      />
    );
    expect(screen.getByText("発送済み")).toBeInTheDocument();
    expect(screen.getByText("ヤマト運輸")).toBeInTheDocument();
    expect(screen.getByText("1234-5678")).toBeInTheDocument();
  });

  it("renders delivered_at when delivered", () => {
    render(
      <ShipmentTrackingBlockPresenter
        status="delivered"
        carrier="佐川急便"
        trackingNumber="SG-1"
        shippedAt="2026-04-01T10:00:00Z"
        deliveredAt="2026-04-03T14:30:00Z"
        labels={baseLabels}
        locale="ja"
      />
    );
    expect(screen.getByText("配達完了")).toBeInTheDocument();
    expect(screen.getByText("配達日時")).toBeInTheDocument();
  });
});
