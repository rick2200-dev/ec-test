"use client";

import { useTranslations } from "next-intl";

type StatusType =
  | "active"
  | "approved"
  | "pending"
  | "suspended"
  | "operational"
  | "degraded"
  | "down";

const statusClassName: Record<StatusType, string> = {
  active: "bg-green-100 text-green-800",
  approved: "bg-green-100 text-green-800",
  pending: "bg-yellow-100 text-yellow-800",
  suspended: "bg-red-100 text-red-800",
  operational: "bg-green-100 text-green-800",
  degraded: "bg-yellow-100 text-yellow-800",
  down: "bg-red-100 text-red-800",
};

const statusKeys: Record<StatusType, string> = {
  active: "status.active",
  approved: "status.approved",
  pending: "status.pending",
  suspended: "status.suspended",
  operational: "status.operational",
  degraded: "status.degraded",
  down: "status.down",
};

export default function StatusBadge({ status }: { status: StatusType }) {
  const t = useTranslations();
  const className = statusClassName[status] ?? "bg-gray-100 text-gray-800";
  const label = statusKeys[status] ? t(statusKeys[status]) : status;
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${className}`}
    >
      {label}
    </span>
  );
}
