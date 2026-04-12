// Re-export the shared UI helpers so existing seller call sites that
// import from `@/lib/utils` keep compiling. New code should import
// directly from `@ec-marketplace/ui-utils`.
export { formatCurrency, STATUS_LABELS, STATUS_COLORS } from "@ec-marketplace/ui-utils";
