import type { Meta, StoryObj } from "@storybook/react";
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
  preparing: "出荷準備中です。発送までしばらくお待ちください",
  carrier: "配送業者",
  trackingNumber: "追跡番号",
  shippedAt: "発送日時",
  deliveredAt: "配達日時",
  statusLabels,
};

const meta: Meta<typeof ShipmentTrackingBlockPresenter> = {
  component: ShipmentTrackingBlockPresenter,
  title: "Buyer/ShipmentTrackingBlock",
};
export default meta;

type Story = StoryObj<typeof ShipmentTrackingBlockPresenter>;

export const Preparing: Story = {
  args: { status: null, labels: baseLabels, locale: "ja" },
};

export const Shipped: Story = {
  args: {
    status: "shipped",
    carrier: "ヤマト運輸",
    trackingNumber: "1234-5678-9012",
    shippedAt: "2026-04-01T10:00:00Z",
    labels: baseLabels,
    locale: "ja",
  },
};

export const Delivered: Story = {
  args: {
    status: "delivered",
    carrier: "佐川急便",
    trackingNumber: "SG-0001",
    shippedAt: "2026-04-01T10:00:00Z",
    deliveredAt: "2026-04-03T14:30:00Z",
    labels: baseLabels,
    locale: "ja",
  },
};
