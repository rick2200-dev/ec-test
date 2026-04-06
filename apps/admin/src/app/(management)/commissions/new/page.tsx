"use client";

import { useState } from "react";
import { tenants, sellers } from "@/lib/mock-data";

export default function NewCommissionRulePage() {
  const [tenantId, setTenantId] = useState("");
  const [sellerId, setSellerId] = useState("");
  const [category, setCategory] = useState("");
  const [rateBasisPoints, setRateBasisPoints] = useState("");
  const [priority, setPriority] = useState("1");
  const [validFrom, setValidFrom] = useState("");
  const [validUntil, setValidUntil] = useState("");

  const ratePercent = rateBasisPoints ? (Number(rateBasisPoints) / 100).toFixed(2) : "0.00";

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    alert("手数料ルールを作成しました（モック）");
  };

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">新規手数料ルール作成</h2>
        <p className="text-text-secondary mt-1">新しい手数料ルールを設定します</p>
      </div>

      <form onSubmit={handleSubmit} className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-6">
        <div>
          <label htmlFor="tenant" className="block text-sm font-medium text-text-primary mb-1">
            テナント <span className="text-danger">*</span>
          </label>
          <select
            id="tenant"
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            required
          >
            <option value="">テナントを選択</option>
            {tenants.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="seller" className="block text-sm font-medium text-text-primary mb-1">
            セラー（任意）
          </label>
          <select
            id="seller"
            value={sellerId}
            onChange={(e) => setSellerId(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
          >
            <option value="">全セラー</option>
            {sellers.map((s) => (
              <option key={s.id} value={s.id}>
                {s.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="category" className="block text-sm font-medium text-text-primary mb-1">
            カテゴリ（任意）
          </label>
          <input
            id="category"
            type="text"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            placeholder="例: 家電"
          />
        </div>

        <div>
          <label htmlFor="rate" className="block text-sm font-medium text-text-primary mb-1">
            手数料率（ベーシスポイント） <span className="text-danger">*</span>
          </label>
          <div className="flex items-center gap-3">
            <input
              id="rate"
              type="number"
              value={rateBasisPoints}
              onChange={(e) => setRateBasisPoints(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              placeholder="例: 1000"
              min="0"
              max="10000"
              required
            />
            <span className="text-sm text-text-secondary whitespace-nowrap">
              = {ratePercent}%
            </span>
          </div>
          <p className="text-xs text-text-secondary mt-1">100ベーシスポイント = 1%</p>
        </div>

        <div>
          <label htmlFor="priority" className="block text-sm font-medium text-text-primary mb-1">
            優先度 <span className="text-danger">*</span>
          </label>
          <input
            id="priority"
            type="number"
            value={priority}
            onChange={(e) => setPriority(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            min="1"
            required
          />
          <p className="text-xs text-text-secondary mt-1">数値が大きいほど優先されます</p>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="validFrom" className="block text-sm font-medium text-text-primary mb-1">
              有効開始日 <span className="text-danger">*</span>
            </label>
            <input
              id="validFrom"
              type="date"
              value={validFrom}
              onChange={(e) => setValidFrom(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              required
            />
          </div>
          <div>
            <label htmlFor="validUntil" className="block text-sm font-medium text-text-primary mb-1">
              有効終了日（任意）
            </label>
            <input
              id="validUntil"
              type="date"
              value={validUntil}
              onChange={(e) => setValidUntil(e.target.value)}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            />
          </div>
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
          >
            作成する
          </button>
          <a
            href="/commissions"
            className="px-4 py-2 bg-surface text-text-primary rounded-lg hover:bg-surface-hover transition-colors text-sm font-medium border border-border"
          >
            キャンセル
          </a>
        </div>
      </form>
    </div>
  );
}
