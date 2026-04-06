"use client";

import { useState } from "react";

export default function NewTenantPage() {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");

  const generateSlug = (value: string) => {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9\u3040-\u309f\u30a0-\u30ff\u4e00-\u9faf]+/g, "-")
      .replace(/^-+|-+$/g, "");
  };

  const handleNameChange = (value: string) => {
    setName(value);
    setSlug(generateSlug(value));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: API call
    alert("テナントを作成しました（モック）");
  };

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">新規テナント作成</h2>
        <p className="text-text-secondary mt-1">新しいテナントを作成します</p>
      </div>

      <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-6">
        <div>
          <label htmlFor="name" className="block text-sm font-medium text-text-primary mb-1">
            テナント名
          </label>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => handleNameChange(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            placeholder="例: 東京マーケット"
            required
          />
        </div>

        <div>
          <label htmlFor="slug" className="block text-sm font-medium text-text-primary mb-1">
            スラッグ
          </label>
          <input
            id="slug"
            type="text"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm font-mono"
            placeholder="例: tokyo-market"
            required
          />
          <p className="text-xs text-text-secondary mt-1">テナント名から自動生成されます</p>
        </div>

        <div>
          <label htmlFor="description" className="block text-sm font-medium text-text-primary mb-1">
            説明
          </label>
          <textarea
            id="description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            rows={3}
            placeholder="テナントの説明を入力"
          />
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
          >
            作成する
          </button>
          <a
            href="/tenants"
            className="px-4 py-2 bg-surface text-text-primary rounded-lg hover:bg-surface-hover transition-colors text-sm font-medium border border-border"
          >
            キャンセル
          </a>
        </div>
      </form>
    </div>
  );
}
