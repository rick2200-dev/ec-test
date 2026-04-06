type StatusType = "active" | "approved" | "pending" | "suspended" | "operational" | "degraded" | "down";

const statusConfig: Record<StatusType, { label: string; className: string }> = {
  active: { label: "有効", className: "bg-green-100 text-green-800" },
  approved: { label: "承認済み", className: "bg-green-100 text-green-800" },
  pending: { label: "承認待ち", className: "bg-yellow-100 text-yellow-800" },
  suspended: { label: "停止中", className: "bg-red-100 text-red-800" },
  operational: { label: "正常", className: "bg-green-100 text-green-800" },
  degraded: { label: "低下", className: "bg-yellow-100 text-yellow-800" },
  down: { label: "停止", className: "bg-red-100 text-red-800" },
};

export default function StatusBadge({ status }: { status: StatusType }) {
  const config = statusConfig[status] ?? { label: status, className: "bg-gray-100 text-gray-800" };
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.className}`}
    >
      {config.label}
    </span>
  );
}
