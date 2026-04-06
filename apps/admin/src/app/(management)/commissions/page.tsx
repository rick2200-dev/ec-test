import { commissionRules } from "@/lib/mock-data";

export default function CommissionsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-text-primary">手数料設定</h2>
          <p className="text-text-secondary mt-1">手数料ルールを管理します</p>
        </div>
        <a
          href="/commissions/new"
          className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
        >
          新規ルール作成
        </a>
      </div>

      <div className="bg-white rounded-lg border border-border shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  テナント
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  セラー
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  カテゴリ
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  手数料率(%)
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  優先度
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  有効期間
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {commissionRules.map((rule) => (
                <tr key={rule.id} className="hover:bg-surface-hover transition-colors">
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {rule.tenantName}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {rule.sellerName ?? "全セラー"}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {rule.category ?? "全カテゴリ"}
                  </td>
                  <td className="px-6 py-4 text-sm font-medium text-text-primary">
                    {rule.rate}%
                  </td>
                  <td className="px-6 py-4 text-sm text-text-primary">
                    {rule.priority}
                  </td>
                  <td className="px-6 py-4 text-sm text-text-secondary">
                    {rule.validFrom} ~ {rule.validUntil ?? "無期限"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
