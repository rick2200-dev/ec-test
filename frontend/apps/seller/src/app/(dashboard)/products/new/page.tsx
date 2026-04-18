"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { categories } from "@/lib/mock-data";
import { ProductForm } from "@/components/ProductForm";
import { SKUManager, SKUInput } from "@/components/SKUManager";
import { useTranslations } from "next-intl";
import { ApiError, fetchAPI } from "@ec-marketplace/api-client";

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export default function NewProductPage() {
  const t = useTranslations();
  const router = useRouter();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [categoryId, setCategoryId] = useState("");
  const [status, setStatus] = useState<"draft" | "active">("draft");
  const [skus, setSkus] = useState<SKUInput[]>([{ code: "", price: "", color: "", size: "" }]);
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

    // Seller identity has to ride in the body until seller-side Auth0
    // wiring lands and the gateway can inject it from JWT context.
    // NEXT_PUBLIC_DEV_SELLER_ID is the dev-only escape hatch.
    const sellerID = process.env.NEXT_PUBLIC_DEV_SELLER_ID;
    if (!sellerID || !UUID_RE.test(sellerID)) {
      setError(
        "NEXT_PUBLIC_DEV_SELLER_ID is not configured (need a UUID until seller auth is wired)."
      );
      setIsSubmitting(false);
      return;
    }

    // Mock categories use "cat-1"-style ids that the catalog service
    // rejects (column is uuid). Drop non-UUID ids so the request succeeds.
    const categoryUUID = categoryId && UUID_RE.test(categoryId) ? categoryId : undefined;

    const body = {
      seller_id: sellerID,
      name,
      slug,
      description,
      ...(categoryUUID ? { category_id: categoryUUID } : {}),
      skus: skus.map((s) => {
        const attrs: Record<string, string> = {};
        if (s.color) attrs.color = s.color;
        if (s.size) attrs.size = s.size;
        return {
          sku_code: s.code,
          price_amount: parseInt(s.price, 10),
          price_currency: "JPY",
          ...(Object.keys(attrs).length ? { attributes: attrs } : {}),
        };
      }),
    };

    try {
      const res = await fetchAPI("/api/v1/seller/products", {
        method: "POST",
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const parsed = (await res.json().catch(() => null)) as {
          error?: string;
          message?: string;
          code?: string;
        } | null;
        const detail = parsed?.error ?? parsed?.message ?? `HTTP ${res.status}`;
        throw new ApiError(res.status, detail, parsed, parsed?.code);
      }

      // Status update is a separate endpoint (PUT /products/{id}/status); the
      // create call always lands in "draft". Honoring the form's status
      // selection requires a follow-up call, which we'll wire when the
      // seller dashboard gets the real product list page.
      void status;

      router.push("/products");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || `エラー: ${err.status}`);
      } else if (err instanceof Error) {
        setError(err.message);
      } else {
        setError(t("newProduct.savedMessage"));
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("newProduct.title")}</h2>
        <p className="text-text-secondary mt-1">{t("newProduct.description")}</p>
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

        <SKUManager skus={skus} onAdd={addSku} onRemove={removeSku} onUpdate={updateSku} />

        {/* Submit */}
        <div className="flex items-center justify-end gap-3">
          <a
            href="/products"
            className="px-4 py-2.5 border border-border rounded-lg text-sm font-medium text-text-primary hover:bg-surface-hover transition-colors"
          >
            {t("newProduct.cancel")}
          </a>
          <button
            type="submit"
            disabled={isSubmitting}
            className="px-6 py-2.5 bg-accent hover:bg-accent-hover text-white rounded-lg text-sm font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting ? t("newProduct.saving") : t("newProduct.save")}
          </button>
        </div>
      </form>
    </div>
  );
}
