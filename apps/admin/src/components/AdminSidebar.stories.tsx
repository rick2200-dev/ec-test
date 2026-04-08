import type { Meta, StoryObj } from "@storybook/react";
import AdminSidebar from "./AdminSidebar";

const meta: Meta<typeof AdminSidebar> = {
  title: "Admin/AdminSidebar",
  component: AdminSidebar,
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <div style={{ height: "100vh" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof AdminSidebar>;

export const Default: Story = {};
