import { inventoryItems } from "@/lib/mock-data";

export default function InventoryPage() {
  const lowStockItems = inventoryItems.filter(
    (item) => item.availableQuantity <= item.lowStockThreshold
  );

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">在庫管理</h2>
        <p className="text-text-secondary mt-1">
          SKUごとの在庫状況を確認・管理できます
        </p>
      </div>

      {/* Low stock alert */}
      {lowStockItems.length > 0 && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 flex items-start gap-3">
          <svg
            className="w-5 h-5 text-danger flex-shrink-0 mt-0.5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"
            />
          </svg>
          <div>
            <p className="text-sm font-medium text-red-800">
              在庫が少ない商品があります（{lowStockItems.length}件）
            </p>
            <p className="text-sm text-red-600 mt-1">
              在庫しきい値（10個以下）を下回っているSKUがあります。早めの補充をおすすめします。
            </p>
          </div>
        </div>
      )}

      {/* Inventory table */}
      <div className="bg-white rounded-lg border border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-border bg-surface">
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  SKU
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  商品名
                </th>
                <th className="text-right px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  在庫数
                </th>
                <th className="text-right px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  予約数
                </th>
                <th className="text-right px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  利用可能数
                </th>
                <th className="text-center px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  在庫アラート
                </th>
                <th className="text-left px-6 py-3 text-xs font-medium text-text-secondary uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {inventoryItems.map((item) => {
                const isLowStock =
                  item.availableQuantity <= item.lowStockThreshold;
                const isOutOfStock = item.availableQuantity <= 0;

                return (
                  <tr
                    key={item.skuId}
                    className={`transition-colors ${
                      isLowStock
                        ? "bg-red-50 hover:bg-red-100"
                        : "hover:bg-surface-hover"
                    }`}
                  >
                    <td className="px-6 py-4 text-sm font-mono text-text-primary">
                      {item.skuCode}
                    </td>
                    <td className="px-6 py-4 text-sm text-text-primary">
                      {item.productName}
                    </td>
                    <td className="px-6 py-4 text-sm text-text-primary text-right">
                      {item.stockQuantity}
                    </td>
                    <td className="px-6 py-4 text-sm text-text-secondary text-right">
                      {item.reservedQuantity}
                    </td>
                    <td className="px-6 py-4 text-right">
                      <span
                        className={`text-sm font-medium ${
                          isOutOfStock
                            ? "text-danger"
                            : isLowStock
                              ? "text-warning"
                              : "text-text-primary"
                        }`}
                      >
                        {item.availableQuantity}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-center">
                      {isOutOfStock ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                          在庫切れ
                        </span>
                      ) : isLowStock ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                          残りわずか
                        </span>
                      ) : (
                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          正常
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4">
                      <button className="text-sm text-accent hover:text-accent-hover font-medium">
                        在庫編集
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
