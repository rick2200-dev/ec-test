import {
  StatusBadgePresenter,
  type StatusBadgePresenterProps,
} from "../../StatusBadge/StatusBadge.presenter";

export interface AdminDashboardStatCard {
  id: string;
  title: string;
  value: string;
  subtitle: string;
}

export interface AdminPendingApplicationRow {
  id: string;
  name: string;
  tenantName: string;
  createdAtLabel: string;
  badge: StatusBadgePresenterProps;
}

export interface AdminServiceHealthRow {
  name: string;
  badge: StatusBadgePresenterProps;
}

export interface AdminDashboardPagePresenterProps {
  heading: { title: string; description: string };
  statsCards: AdminDashboardStatCard[];
  pendingSection: {
    title: string;
    viewAllHref: string;
    viewAllLabel: string;
    columnLabels: {
      sellerName: string;
      tenant: string;
      applicationDate: string;
      status: string;
    };
    rows: AdminPendingApplicationRow[];
  };
  serviceHealthSection: {
    title: string;
    services: AdminServiceHealthRow[];
  };
}

export function AdminDashboardPagePresenter({
  heading,
  statsCards,
  pendingSection,
  serviceHealthSection,
}: AdminDashboardPagePresenterProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{heading.title}</h2>
        <p className="text-text-secondary mt-1">{heading.description}</p>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statsCards.map((card) => (
          <div key={card.id} className="bg-white rounded-lg border border-border p-6 shadow-sm">
            <p className="text-sm text-text-secondary">{card.title}</p>
            <p className="text-2xl font-bold text-text-primary mt-1">{card.value}</p>
            <p className="text-xs text-text-secondary mt-2">{card.subtitle}</p>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Pending seller applications */}
        <div className="bg-white rounded-lg border border-border shadow-sm">
          <div className="px-6 py-4 border-b border-border flex items-center justify-between">
            <h3 className="text-lg font-semibold text-text-primary">{pendingSection.title}</h3>
            <a
              href={pendingSection.viewAllHref}
              className="text-sm text-accent hover:text-accent-hover font-medium"
            >
              {pendingSection.viewAllLabel} &rarr;
            </a>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-border bg-surface">
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {pendingSection.columnLabels.sellerName}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {pendingSection.columnLabels.tenant}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {pendingSection.columnLabels.applicationDate}
                  </th>
                  <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                    {pendingSection.columnLabels.status}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {pendingSection.rows.map((row) => (
                  <tr key={row.id} className="hover:bg-surface-hover transition-colors">
                    <td className="px-6 py-4 text-sm font-medium text-text-primary">{row.name}</td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{row.tenantName}</td>
                    <td className="px-6 py-4 text-sm text-text-secondary">{row.createdAtLabel}</td>
                    <td className="px-6 py-4">
                      <StatusBadgePresenter {...row.badge} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Platform health */}
        <div className="bg-white rounded-lg border border-border shadow-sm">
          <div className="px-6 py-4 border-b border-border">
            <h3 className="text-lg font-semibold text-text-primary">{serviceHealthSection.title}</h3>
          </div>
          <div className="p-6 space-y-4">
            {serviceHealthSection.services.map((service) => (
              <div key={service.name} className="flex items-center justify-between">
                <span className="text-sm text-text-primary">{service.name}</span>
                <StatusBadgePresenter {...service.badge} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
