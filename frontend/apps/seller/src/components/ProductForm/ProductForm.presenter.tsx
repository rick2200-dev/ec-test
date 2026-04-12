"use client";

import type { Category } from "../../lib/types";

export interface ProductFormPresenterProps {
  name: string;
  slug: string;
  description: string;
  categoryId: string;
  status: "draft" | "active";
  categories: Category[];
  onNameChange: (value: string) => void;
  onSlugChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onCategoryChange: (value: string) => void;
  onStatusChange: (value: "draft" | "active") => void;
}

export function ProductFormPresenter({
  name,
  slug,
  description,
  categoryId,
  status,
  categories,
  onNameChange,
  onSlugChange,
  onDescriptionChange,
  onCategoryChange,
  onStatusChange,
}: ProductFormPresenterProps) {
  return (
    <div className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-4">
      <h3 className="text-lg font-semibold text-text-primary">基本情報</h3>

      <div>
        <label htmlFor="product-name" className="block text-sm font-medium text-text-primary mb-1">
          商品名 <span className="text-danger">*</span>
        </label>
        <input
          id="product-name"
          type="text"
          value={name}
          onChange={(e) => onNameChange(e.target.value)}
          placeholder="例: オーガニックコットンTシャツ"
          className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
          required
          aria-required="true"
        />
      </div>

      <div>
        <label htmlFor="product-slug" className="block text-sm font-medium text-text-primary mb-1">
          スラッグ
        </label>
        <input
          id="product-slug"
          type="text"
          value={slug}
          onChange={(e) => onSlugChange(e.target.value)}
          placeholder="auto-generated-slug"
          className="w-full px-3 py-2.5 border border-border rounded-lg text-sm bg-surface text-text-secondary focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
        />
        <p className="text-xs text-text-secondary mt-1">
          商品名から自動生成されます。変更も可能です。
        </p>
      </div>

      <div>
        <label
          htmlFor="product-description"
          className="block text-sm font-medium text-text-primary mb-1"
        >
          説明
        </label>
        <textarea
          id="product-description"
          value={description}
          onChange={(e) => onDescriptionChange(e.target.value)}
          rows={4}
          placeholder="商品の説明を入力してください..."
          className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent resize-y"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label
            htmlFor="product-category"
            className="block text-sm font-medium text-text-primary mb-1"
          >
            カテゴリ <span className="text-danger">*</span>
          </label>
          <select
            id="product-category"
            value={categoryId}
            onChange={(e) => onCategoryChange(e.target.value)}
            className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
            required
            aria-required="true"
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
          <label
            htmlFor="product-status"
            className="block text-sm font-medium text-text-primary mb-1"
          >
            ステータス
          </label>
          <select
            id="product-status"
            value={status}
            onChange={(e) => onStatusChange(e.target.value as "draft" | "active")}
            className="w-full px-3 py-2.5 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/20 focus:border-accent"
          >
            <option value="draft">下書き</option>
            <option value="active">公開</option>
          </select>
        </div>
      </div>
    </div>
  );
}
