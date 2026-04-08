"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { categories } from "@/lib/mock-data";
import { ProductForm } from "@/components/ProductForm";
import { SKUManager, SKUInput } from "@/components/SKUManager";

export default function NewProductPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState<"draft" | "active">("draft");
  const [skus, setSkus] = useState<SKUInput[]>([
    { code: "", price: "", color: "", size: "" },
  ]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      const res = await fetch("/api/products", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name,
          slug,
          description,
          category_id: categoryId,
          status,
          skus: skus.map((s) => ({
            code: s.code,
            price: parseInt(s.price, 10),
            attributes: {
              ...(s.color ? { color: s.color } : {}),
              ...(s.size ? { size: s.size } : {}),
            },
          })),
        }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error ?? `エラー: ${res.status}`);
      }

      router.push("/products");
    } catch (err) {
      setError(err instanceof Error ? err.message : "商品の保存に失敗しました");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">新規商品登録</h2>
        <p className="text-text-secondary mt-1">
          商品情報を入力して登録してください
        </p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg px-4 py-3 text-sm text-danger">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        <ProductForm
          name={name}
          slug={slug}
          description={description}
          categoryId={categoryId}
          status={status}
          categories={categories}
          onNameChange={handleNameChange}
          onSlugChange={setSlug}
          onDescriptionChange={setDescription}
          onCategoryChange={setCategoryId}
          onStatusChange={setStatus}
        />

        <SKUManager
          skus={skus}
          onAdd={addSku}
          onRemove={removeSku}
          onUpdate={updateSku}
        />

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
            disabled={isSubmitting}
            className="px-6 py-2.5 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting ? "保存中..." : "保存する"}
          </button>
        </div>
      </form>
    </div>
  );
}
