import type { Meta, StoryObj } from "@storybook/react";
import type { Shipment, ShipmentStatus } from "@ec-marketplace/types";
import { ShipmentsListPresenter } from "./ShipmentsList.presenter";
import { RegisterTrackingModalPresenter } from "./RegisterTrackingModal.presenter";

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

const sample = (overrides: Partial<Shipment>): Shipment => ({
  id: "11111111-1111-1111-1111-111111111111",
  seller_id: "22222222-2222-2222-2222-222222222222",
  order_id: "33333333-3333-3333-3333-333333333333",
  buyer_auth0_id: "auth0|1",
  status: "ready_to_ship",
  created_at: "2026-04-01T10:00:00Z",
  updated_at: "2026-04-01T10:00:00Z",
  ...overrides,
});

const meta: Meta<typeof ShipmentsListPresenter> = {
  component: ShipmentsListPresenter,
  title: "Pages/ShipmentsPage",
};
export default meta;

type Story = StoryObj<typeof ShipmentsListPresenter>;

export const Empty: Story = {
  args: {
    shipments: [],
    emptyLabel: "対象の出荷はありません",
    loadingLabel: "...",
    loaded: true,
    statusLabels,
    columns,
    renderActions: () => null,
    locale: "ja",
  },
};

export const MixedStatuses: Story = {
  args: {
    loaded: true,
    emptyLabel: "",
    loadingLabel: "",
    statusLabels,
    columns,
    locale: "ja",
    shipments: [
      sample({ status: "ready_to_ship" }),
      sample({
        id: "44444444-4444-4444-4444-444444444444",
        status: "shipped",
        carrier: "ヤマト運輸",
        tracking_number: "1234",
        shipped_at: "2026-04-02T09:30:00Z",
      }),
      sample({
        id: "55555555-5555-5555-5555-555555555555",
        status: "delivered",
        carrier: "佐川急便",
        tracking_number: "5678",
        shipped_at: "2026-03-28T15:00:00Z",
        delivered_at: "2026-03-31T11:00:00Z",
      }),
    ],
    renderActions: () => (
      <span className="text-xs text-text-secondary">
        {/* container injects real action buttons */}
        actions
      </span>
    ),
  },
};

export const RegisterModalIdle: StoryObj<typeof RegisterTrackingModalPresenter> = {
  render: (args) => <RegisterTrackingModalPresenter {...args} />,
  args: {
    open: true,
    carrier: "",
    trackingNumber: "",
    note: "",
    submitting: false,
    onCarrierChange: () => {},
    onTrackingNumberChange: () => {},
    onNoteChange: () => {},
    onSubmit: () => {},
    onClose: () => {},
    labels: {
      title: "出荷登録",
      carrier: "配送業者",
      trackingNumber: "追跡番号",
      note: "メモ",
      submit: "登録する",
      submitting: "登録中...",
      cancel: "キャンセル",
    },
  },
};
