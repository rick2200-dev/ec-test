import type { Meta, StoryObj } from "@storybook/react";
import StatusBadge from "./StatusBadge";

const meta: Meta<typeof StatusBadge> = {
  title: "Admin/StatusBadge",
  component: StatusBadge,
  parameters: { layout: "centered" },
};

export default meta;
type Story = StoryObj<typeof StatusBadge>;

export const Active: Story = { args: { status: "active" } };
export const Approved: Story = { args: { status: "approved" } };
export const Pending: Story = { args: { status: "pending" } };
export const Suspended: Story = { args: { status: "suspended" } };
export const Operational: Story = { args: { status: "operational" } };
export const Degraded: Story = { args: { status: "degraded" } };
export const Down: Story = { args: { status: "down" } };
