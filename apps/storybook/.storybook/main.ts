import type { StorybookConfig } from "@storybook/react-vite";
import path from "node:path";

const config: StorybookConfig = {
  stories: [
    "../../buyer/src/components/**/*.stories.@(ts|tsx)",
    "../../seller/src/components/**/*.stories.@(ts|tsx)",
    "../../admin/src/components/**/*.stories.@(ts|tsx)",
  ],
  addons: ["@storybook/addon-essentials", "@storybook/addon-a11y", "@storybook/addon-interactions"],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  viteFinal: async (config) => {
    config.resolve = config.resolve || {};
    config.resolve.alias = {
      ...config.resolve.alias,
      "@/lib/mock-data": path.resolve(__dirname, "../../buyer/src/lib/mock-data"),
      "@/lib/types": path.resolve(__dirname, "../../buyer/src/lib/types"),
      "@/lib/api": path.resolve(__dirname, "../../buyer/src/lib/api"),
      "@/components": path.resolve(__dirname, "../../buyer/src/components"),
    };
    return config;
  },
};

export default config;
