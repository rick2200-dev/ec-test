export interface StatusBadgePresenterProps {
  /** Tone determines the visual color of the badge. */
  tone: "success" | "warning" | "danger" | "neutral";
  /** Pre-localized label to display inside the badge. */
  label: string;
}

const toneClassName: Record<StatusBadgePresenterProps["tone"], string> = {
  success: "bg-green-100 text-green-800",
  warning: "bg-yellow-100 text-yellow-800",
  danger: "bg-red-100 text-red-800",
  neutral: "bg-gray-100 text-gray-800",
};

export function StatusBadgePresenter({ tone, label }: StatusBadgePresenterProps) {
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${toneClassName[tone]}`}
    >
      {label}
    </span>
  );
}
