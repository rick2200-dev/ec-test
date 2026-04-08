import js from "@eslint/js";
import tseslint from "typescript-eslint";
import prettierConfig from "eslint-config-prettier";

/** @type {import("eslint").Linter.Config[]} */
export default [
  js.configs.recommended,
  ...tseslint.configs.recommended,
  prettierConfig,
  {
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/no-explicit-any": "warn",
    },
  },
  {
    ignores: [
      "node_modules/",
      ".next/",
      "dist/",
      "coverage/",
      ".turbo/",
      "*.config.ts",
      "*.config.mjs",
      "*.setup.ts",
    ],
  },
];
