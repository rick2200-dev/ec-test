/** Formats an amount in JPY with locale-aware thousands separator. */
export function formatCurrency(amount: number): string {
  return `¥${amount.toLocaleString()}`;
}

/** Display labels for order and product statuses. */
export const STATUS_LABELS: Record<string, string> = {
  pending: "未処理",
  processing: "処理中",
  shipped: "発送済み",
  delivered: "配達済み",
  completed: "完了",
  cancelled: "キャンセル",
  draft: "下書き",
  active: "公開中",
  archived: "アーカイブ",
};

/** Tailwind color classes for order and product statuses. */
export const STATUS_COLORS: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-800",
  processing: "bg-blue-100 text-blue-800",
  shipped: "bg-purple-100 text-purple-800",
  delivered: "bg-indigo-100 text-indigo-800",
  completed: "bg-green-100 text-green-800",
  cancelled: "bg-red-100 text-red-800",
  draft: "bg-gray-100 text-gray-700",
  active: "bg-green-100 text-green-800",
  archived: "bg-gray-100 text-gray-500",
};
