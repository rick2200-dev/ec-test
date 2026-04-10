import type { StorybookConfig } from "@storybook/react-vite";
import path from "node:path";

const config: StorybookConfig = {
  // The recursive `**/` glob picks up component stories AND page-level
  // presenter stories under e.g. `components/pages/HomePage/`.
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
  // Path-alias notes:
  // - Stories should always import the *.presenter.tsx file (pure, props-only)
  //   so they have no data-fetching dependencies.
  // - The buyer aliases below remain because each `index.ts` barrel re-exports
  //   the Container alongside the Presenter, which forces Vite to resolve
  //   (but not execute) the Container's `@/lib/...` imports during bundling.
  // - Seller and admin presenters intentionally avoid `@/` and use relative
  //   imports (`../../lib/types`, `../../StatsCard/StatsCard.presenter`, etc.),
  //   so no per-app aliases are needed for them. Do not add cross-component
  //   `@/` imports inside seller/admin presenters — they would silently
  //   resolve to the buyer paths below.
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
