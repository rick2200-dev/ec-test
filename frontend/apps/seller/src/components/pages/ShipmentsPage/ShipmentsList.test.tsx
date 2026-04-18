import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { Shipment, ShipmentStatus } from "@ec-marketplace/types";
import { ShipmentsListPresenter } from "./ShipmentsList.presenter";

const statusLabels: Record<ShipmentStatus, string> = {
  pending: "未準備",
  ready_to_ship: "出荷準備完了",
  shipped: "発送済み",
  delivered: "配達完了",
  cancelled: "キャンセル",
};

const columns = {
  orderId: "注文",
  status: "状態",
  carrier: "配送業者",
  tracking: "追跡番号",
  shippedAt: "発送日時",
  actions: "操作",
};

const sample: Shipment = {
  id: "11111111-1111-1111-1111-111111111111",
  seller_id: "22222222-2222-2222-2222-222222222222",
  order_id: "33333333-3333-3333-3333-333333333333",
  buyer_auth0_id: "auth0|1",
  status: "shipped",
  carrier: "ヤマト運輸",
  tracking_number: "1234-5678-9012",
  shipped_at: "2026-04-01T12:00:00Z",
  created_at: "2026-04-01T10:00:00Z",
  updated_at: "2026-04-01T12:00:00Z",
};

describe("ShipmentsListPresenter", () => {
  it("renders the loading state", () => {
    render(
      <ShipmentsListPresenter
        shipments={[]}
        emptyLabel="該当なし"
        loadingLabel="読み込み中..."
        loaded={false}
        statusLabels={statusLabels}
        columns={columns}
        renderActions={() => null}
        locale="ja"
      />
    );
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
  });

  it("renders the empty state when loaded with no rows", () => {
    render(
      <ShipmentsListPresenter
        shipments={[]}
        emptyLabel="該当なし"
        loadingLabel=""
        loaded
        statusLabels={statusLabels}
        columns={columns}
        renderActions={() => null}
        locale="ja"
      />
    );
    expect(screen.getByText("該当なし")).toBeInTheDocument();
  });

  it("renders shipment rows with status badge and actions slot", () => {
    render(
      <ShipmentsListPresenter
        shipments={[sample]}
        emptyLabel=""
        loadingLabel=""
        loaded
        statusLabels={statusLabels}
        columns={columns}
        renderActions={(s) => <span data-testid={`action-${s.id}`}>action</span>}
        locale="ja"
      />
    );
    expect(screen.getByText("ヤマト運輸")).toBeInTheDocument();
    expect(screen.getByText("1234-5678-9012")).toBeInTheDocument();
    expect(screen.getByText("発送済み")).toBeInTheDocument();
    expect(screen.getByTestId(`action-${sample.id}`)).toBeInTheDocument();
  });
});
