"use client";

import { useState } from "react";
import { categories } from "@/lib/mock-data";

interface SKUInput {
  code: string;
  price: string;
  color: string;
  size: string;
}

export default function NewProductPage() {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState<"draft" | "active">("draft");
  const [skus, setSkus] = useState<SKUInput[]>([
    { code: "", price: "", color: "", size: "" },
  ]);

  const handleNameChange = (value: string) => {
    setName(value);
    const generated = value
      .toLowerCase()
      .replace(/[^\w\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .trim();
    setSlug(generated);
  };

  const addSku = () => {
    setSkus([...skus, { code: "", price: "", color: "", size: "" }]);
  };

  const removeSku = (index: number) => {
    if (skus.length <= 1) return;
    setSkus(skus.filter((_, i) => i !== index));
  };

  const updateSku = (index: number, field: keyof SKUInput, value: string) => {
    const updated = [...skus];
    updated[index] = { ...updated[index], [field]: value };
    setSkus(updated);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: API call to create product
    alert("商品を保存しました（デモ）");
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">新規商品登録</h2>
        <p className="text-text-secondary mt-1">
          商品情報を入力して登録してください
        </p>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic info */}
        <div className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-4">
          <h3 className="text-lg font-semibold text-text-primary">基本情報</h3>

          <div>
            <label className="block text-sm font-medium text-text-primary mb-1">
              商品名 <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => handleNameChange(e.target.value)}
              placeholder="例: オーガニックコットンTシャツ"
              className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
              required
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-text-primary mb-1">
              スラッグ
            </label>
            <input
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="auto-generated-slug"
              className="w-full px-3 py-2.5 border border-border rounded-lg text-sm bg-surface text-text-secondary focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
            />
            <p className="text-xs text-text-secondary mt-1">
              商品名から自動生成されます。変更も可能です。
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-text-primary mb-1">
              説明
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              placeholder="商品の説明を入力してください..."
              className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent resize-y"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-text-primary mb-1">
                カテゴリ <span className="text-danger">*</span>
              </label>
              <select
                value={categoryId}
                onChange={(e) => setCategoryId(e.target.value)}
                className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                required
              >
                <option value="">カテゴリを選択</option>
                {categories.map((cat) => (
                  <option key={cat.id} value={cat.id}>
                    {cat.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-text-primary mb-1">
                ステータス
              </label>
              <select
                value={status}
                onChange={(e) =>
                  setStatus(e.target.value as "draft" | "active")
                }
                className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
              >
                <option value="draft">下書き</option>
                <option value="active">公開</option>
              </select>
            </div>
          </div>
        </div>

        {/* SKUs */}
        <div className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-text-primary">
              SKU（バリエーション）
            </h3>
            <button
              type="button"
              onClick={addSku}
              className="inline-flex items-center gap-1 text-sm text-accent hover:text-accent-hover font-medium"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              SKUを追加
            </button>
          </div>

          {skus.map((sku, index) => (
            <div
              key={index}
              className="border border-border rounded-lg p-4 space-y-3"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-text-secondary">
                  SKU #{index + 1}
                </span>
                {skus.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeSku(index)}
                    className="text-sm text-danger hover:text-red-700 font-medium"
                  >
                    削除
                  </button>
                )}
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-text-secondary mb-1">
                    SKUコード <span className="text-danger">*</span>
                  </label>
                  <input
                    type="text"
                    value={sku.code}
                    onChange={(e) => updateSku(index, "code", e.target.value)}
                    placeholder="例: OCT-WHT-M"
                    className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                    required
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-text-secondary mb-1">
                    価格（税込） <span className="text-danger">*</span>
                  </label>
                  <input
                    type="number"
                    value={sku.price}
                    onChange={(e) => updateSku(index, "price", e.target.value)}
                    placeholder="例: 3980"
                    className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                    required
                    min="0"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-text-secondary mb-1">
                    カラー
                  </label>
                  <input
                    type="text"
                    value={sku.color}
                    onChange={(e) => updateSku(index, "color", e.target.value)}
                    placeholder="例: ホワイト"
                    className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-text-secondary mb-1">
                    サイズ
                  </label>
                  <input
                    type="text"
                    value={sku.size}
                    onChange={(e) => updateSku(index, "size", e.target.value)}
                    placeholder="例: M"
                    className="w-full px-3 py-2 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
                  />
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Submit */}
        <div className="flex items-center justify-end gap-3">
          <a
            href="/products"
            className="px-4 py-2.5 border border-border rounded-lg text-sm font-medium text-text-primary hover:bg-surface-hover transition-colors"
          >
            キャンセル
          </a>
          <button
            type="submit"
            className="px-6 py-2.5 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors"
          >
            保存する
          </button>
        </div>
      </form>
    </div>
  );
}
