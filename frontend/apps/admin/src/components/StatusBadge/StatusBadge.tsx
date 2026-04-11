"use client";

import { useTranslations } from "next-intl";
import { StatusBadgePresenter, type StatusBadgePresenterProps } from "./StatusBadge.presenter";

export type StatusType =
  | "active"
  | "archived"
  | "approved"
  | "pending"
  | "suspended"
  | "operational"
  | "degraded"
  | "down";

const statusTone: Record<StatusType, StatusBadgePresenterProps["tone"]> = {
  active: "success",
  archived: "neutral",
  approved: "success",
  pending: "warning",
  suspended: "danger",
  operational: "success",
  degraded: "warning",
  down: "danger",
};

const statusKeys: Record<StatusType, string> = {
  active: "status.active",
  archived: "status.archived",
  approved: "status.approved",
  pending: "status.pending",
  suspended: "status.suspended",
  operational: "status.operational",
  degraded: "status.degraded",
  down: "status.down",
};

export default function StatusBadge({ status }: { status: StatusType }) {
  const t = useTranslations();
  const tone = statusTone[status] ?? "neutral";
  const label = statusKeys[status] ? t(statusKeys[status]) : status;
  return <StatusBadgePresenter tone={tone} label={label} />;
}
