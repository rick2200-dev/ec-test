/**
 * Display labels and Tailwind color classes for order + product
 * statuses. The keys intentionally cover both order statuses
 * (pending/processing/shipped/...) and product statuses
 * (draft/active/archived) so callers can key directly off whatever
 * status string they have.
 */

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
