interface StatsCardProps {
  title: string;
  value: string;
  subtitle?: string;
  trend?: "up" | "down" | "neutral";
  accent?: "default" | "success" | "warning" | "danger";
}

const accentColors = {
  default: "border-l-accent",
  success: "border-l-success",
  warning: "border-l-warning",
  danger: "border-l-danger",
};

export default function StatsCard({
  title,
  value,
  subtitle,
  accent = "default",
}: StatsCardProps) {
  return (
    <div
      className={`bg-white rounded-lg border border-border border-l-4 ${accentColors[accent]} p-6 shadow-sm`}
    >
      <p className="text-sm text-text-secondary font-medium">{title}</p>
      <p className="text-2xl font-bold text-text-primary mt-1">{value}</p>
      {subtitle && (
        <p className="text-xs text-text-secondary mt-2">{subtitle}</p>
      )}
    </div>
  );
}
