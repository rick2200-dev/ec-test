"use client";

import { useTranslations } from "next-intl";
import { StatusBadgePresenter, type StatusBadgePresenterProps } from "./StatusBadge.presenter";

export type StatusType =
  | "active"
  | "approved"
  | "pending"
  | "suspended"
  | "operational"
  | "degraded"
  | "down"
  | "archived";

const statusTone: Record<StatusType, StatusBadgePresenterProps["tone"]> = {
  active: "success",
  approved: "success",
  pending: "warning",
  suspended: "danger",
  operational: "success",
  degraded: "warning",
  down: "danger",
  archived: "neutral",
};

const statusKeys: Record<StatusType, string> = {
  active: "status.active",
  approved: "status.approved",
  pending: "status.pending",
  suspended: "status.suspended",
  operational: "status.operational",
  degraded: "status.degraded",
  down: "status.down",
  archived: "status.archived",
};

export default function StatusBadge({ status }: { status: StatusType }) {
  const t = useTranslations();
  const tone = statusTone[status] ?? "neutral";
  const label = statusKeys[status] ? t(statusKeys[status]) : status;
  return <StatusBadgePresenter tone={tone} label={label} />;
}
