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
  // - Stories AND presenters must import sibling presenters via the
  //   `*.presenter.tsx` file directly — never via the folder `index.ts`
  //   barrel. Barrels re-export the Container, which pulls in data/i18n
  //   modules (e.g. `lib/api.ts` reads `process.env.NEXT_PUBLIC_API_URL`
  //   at module top-level). Vite does not replace `process.env.X` like
  //   Next.js does, so a single barrel import can crash the entire UI
  //   with `process is not defined`.
  // - The buyer `@/` aliases below remain as a safety net so that if a
  //   barrel does sneak in, Vite can still *resolve* the alias (the
  //   `define` block below then prevents `process.env` from blowing up).
  // - Seller and admin presenters intentionally avoid `@/` and use
  //   relative imports (`../../lib/types`, `../../StatsCard/StatsCard.presenter`,
  //   etc.). Do not add cross-component `@/` imports inside seller/admin
  //   presenters — they would silently resolve to the buyer paths below.
  viteFinal: async (config) => {
    config.resolve = config.resolve || {};
    config.resolve.alias = {
      ...config.resolve.alias,
      "@/lib/mock-data": path.resolve(__dirname, "../../buyer/src/lib/mock-data"),
      "@/lib/types": path.resolve(__dirname, "../../buyer/src/lib/types"),
      "@/lib/api": path.resolve(__dirname, "../../buyer/src/lib/api"),
      "@/components": path.resolve(__dirname, "../../buyer/src/components"),
    };
    // Defense-in-depth: stub out `process.env` so any stray Next.js-style
    // env access in transitively-bundled modules doesn't crash the browser
    // bundle. `api.ts` falls back to a default URL when this is undefined.
    //
    // The value must be an esbuild-compatible JSON literal. `JSON.stringify({})`
    // produces the string `"{}"`, which esbuild accepts as an empty-object
    // literal. Raw `"({})"` is rejected by newer esbuild versions with
    // `Invalid define value (must be an entity name or JS literal)`.
    config.define = {
      ...config.define,
      "process.env": JSON.stringify({}),
    };
    // Force the automatic JSX runtime in esbuild. The shared monorepo
    // tsconfig sets `jsx: "preserve"` for Next.js, which would otherwise
    // make esbuild leave JSX untouched and surface "React is not defined"
    // at runtime. The automatic runtime injects `_jsx`/`_jsxs` imports
    // from `react/jsx-runtime`, so presenters never need to import React.
    config.esbuild = {
      ...config.esbuild,
      jsx: "automatic",
      jsxImportSource: "react",
    };
    return config;
  },
};

export default config;
