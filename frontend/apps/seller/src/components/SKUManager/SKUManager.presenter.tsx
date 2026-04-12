"use client";

export interface SKUInput {
  code: string;
  price: string;
  color: string;
  size: string;
}

export interface SKUManagerPresenterProps {
  skus: SKUInput[];
  onAdd: () => void;
  onRemove: (index: number) => void;
  onUpdate: (index: number, field: keyof SKUInput, value: string) => void;
}

export function SKUManagerPresenter({ skus, onAdd, onRemove, onUpdate }: SKUManagerPresenterProps) {
  return (
    <div className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-text-primary">SKU（バリエーション）</h3>
        <button
          type="button"
          onClick={onAdd}
          className="inline-flex items-center gap-1 text-sm text-accent hover:text-accent-hover font-medium"
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          SKUを追加
        </button>
      </div>

      {skus.map((sku, index) => (
        <div key={index} className="border border-border rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-text-secondary">SKU #{index + 1}</span>
            {skus.length > 1 && (
              <button
                type="button"
                onClick={() => onRemove(index)}
                className="text-sm text-danger hover:text-red-700 font-medium"
              >
                削除
              </button>
            )}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label
                htmlFor={`sku-code-${index}`}
                className="block text-xs font-medium text-text-secondary mb-1"
              >
                SKUコード <span className="text-danger">*</span>
              </label>
              <input
                id={`sku-code-${index}`}
                type="text"
                value={sku.code}
                onChange={(e) => onUpdate(index, "code", e.target.value)}
                placeholder="例: OCT-WHT-M"
                className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                required
                aria-required="true"
              />
            </div>
            <div>
              <label
                htmlFor={`sku-price-${index}`}
                className="block text-xs font-medium text-text-secondary mb-1"
              >
                価格（税込） <span className="text-danger">*</span>
              </label>
              <input
                id={`sku-price-${index}`}
                type="number"
                value={sku.price}
                onChange={(e) => onUpdate(index, "price", e.target.value)}
                placeholder="例: 3980"
                className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                required
                aria-required="true"
                min="0"
              />
            </div>
            <div>
              <label
                htmlFor={`sku-color-${index}`}
                className="block text-xs font-medium text-text-secondary mb-1"
              >
                カラー
              </label>
              <input
                id={`sku-color-${index}`}
                type="text"
                value={sku.color}
                onChange={(e) => onUpdate(index, "color", e.target.value)}
                placeholder="例: ホワイト"
                className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
              />
            </div>
            <div>
              <label
                htmlFor={`sku-size-${index}`}
                className="block text-xs font-medium text-text-secondary mb-1"
              >
                サイズ
              </label>
              <input
                id={`sku-size-${index}`}
                type="text"
                value={sku.size}
                onChange={(e) => onUpdate(index, "size", e.target.value)}
                placeholder="例: M"
                className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
              />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
