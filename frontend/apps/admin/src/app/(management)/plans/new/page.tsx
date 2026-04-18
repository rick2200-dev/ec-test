"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import { createAdminPlan } from "@/lib/api";

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export default function NewPlanPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [tier, setTier] = useState(0);
  const [priceAmount, setPriceAmount] = useState(0);
  const [searchBoost, setSearchBoost] = useState(1.0);
  const [maxProducts, setMaxProducts] = useState(10);
  const [featuredSlots, setFeaturedSlots] = useState(0);
  const [promotedResults, setPromotedResults] = useState(0);
  const [stripePriceId, setStripePriceId] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const t = useTranslations();

  const generateSlug = (value: string) => {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-+|-+$/g, "");
  };

  const handleNameChange = (value: string) => {
    setName(value);
    setSlug(generateSlug(value));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (submitting) return;
    // Admin plans belong to a tenant; the gateway doesn't inject
    // tenant_id for platform admins yet, so the form reads it from env.
    // Replace with a tenant picker once the admin app has auth.
    const tenantId = process.env.NEXT_PUBLIC_DEV_TENANT_ID;
    if (!tenantId || !UUID_RE.test(tenantId)) {
      setError("NEXT_PUBLIC_DEV_TENANT_ID must be a UUID in .env.local");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createAdminPlan({
        tenant_id: tenantId,
        name,
        slug,
        tier,
        price_amount: priceAmount,
        price_currency: "JPY",
        features: {
          max_products: maxProducts,
          search_boost: searchBoost,
          featured_slots: featuredSlots,
          promoted_results: promotedResults,
        },
        stripe_price_id: stripePriceId,
      });
      router.push("/plans");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || t("newPlan.errorGeneric"));
      } else {
        setError(t("newPlan.errorGeneric"));
      }
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("newPlan.title")}</h2>
        <p className="text-text-secondary mt-1">{t("newPlan.description")}</p>
      </div>

      {error && (
        <div
          role="alert"
          className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-danger"
        >
          {error}
        </div>
      )}

      <form
        onSubmit={handleSubmit}
        className="bg-white rounded-lg border border-border shadow-sm p-6 space-y-6"
      >
        <div>
          <label htmlFor="name" className="block text-sm font-medium text-text-primary mb-1">
            {t("newPlan.planName")}
          </label>
          <input
            id="name"
            type="text"
            value={name}
            onChange={(e) => handleNameChange(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
            required
            aria-required="true"
          />
        </div>

        <div>
          <label htmlFor="slug" className="block text-sm font-medium text-text-primary mb-1">
            {t("newPlan.slug")}
          </label>
          <input
            id="slug"
            type="text"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm font-mono"
            required
            aria-required="true"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label htmlFor="tier" className="block text-sm font-medium text-text-primary mb-1">
              {t("newPlan.tier")}
            </label>
            <input
              id="tier"
              type="number"
              value={tier}
              onChange={(e) => setTier(Number(e.target.value))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              min={0}
              required
              aria-required="true"
            />
          </div>

          <div>
            <label
              htmlFor="priceAmount"
              className="block text-sm font-medium text-text-primary mb-1"
            >
              {t("newPlan.priceAmount")}
            </label>
            <input
              id="priceAmount"
              type="number"
              value={priceAmount}
              onChange={(e) => setPriceAmount(Number(e.target.value))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              min={0}
              required
              aria-required="true"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="searchBoost"
              className="block text-sm font-medium text-text-primary mb-1"
            >
              {t("newPlan.searchBoost")}
            </label>
            <input
              id="searchBoost"
              type="number"
              value={searchBoost}
              onChange={(e) => setSearchBoost(Number(e.target.value))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              step={0.1}
              min={0.1}
              required
              aria-required="true"
            />
          </div>

          <div>
            <label
              htmlFor="maxProducts"
              className="block text-sm font-medium text-text-primary mb-1"
            >
              {t("newPlan.maxProducts")}
            </label>
            <input
              id="maxProducts"
              type="number"
              value={maxProducts}
              onChange={(e) => setMaxProducts(Number(e.target.value))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              min={-1}
              required
              aria-required="true"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label
              htmlFor="featuredSlots"
              className="block text-sm font-medium text-text-primary mb-1"
            >
              {t("newPlan.featuredSlots")}
            </label>
            <input
              id="featuredSlots"
              type="number"
              value={featuredSlots}
              onChange={(e) => setFeaturedSlots(Number(e.target.value))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              min={0}
              required
              aria-required="true"
            />
          </div>

          <div>
            <label
              htmlFor="promotedResults"
              className="block text-sm font-medium text-text-primary mb-1"
            >
              {t("newPlan.promotedResults")}
            </label>
            <input
              id="promotedResults"
              type="number"
              value={promotedResults}
              onChange={(e) => setPromotedResults(Number(e.target.value))}
              className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm"
              min={0}
              required
              aria-required="true"
            />
          </div>
        </div>

        <div>
          <label
            htmlFor="stripePriceId"
            className="block text-sm font-medium text-text-primary mb-1"
          >
            {t("newPlan.stripePriceId")}
          </label>
          <input
            id="stripePriceId"
            type="text"
            value={stripePriceId}
            onChange={(e) => setStripePriceId(e.target.value)}
            className="w-full px-3 py-2 border border-border rounded-lg focus:outline-none focus:ring-2 focus:ring-accent text-sm font-mono"
            placeholder="price_xxxxx"
          />
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={submitting}
            className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium disabled:opacity-50"
          >
            {submitting ? t("newPlan.creating") : t("newPlan.create")}
          </button>
          <a
            href="/plans"
            className="px-4 py-2 bg-surface text-text-primary rounded-lg hover:bg-surface-hover transition-colors text-sm font-medium border border-border"
          >
            {t("newPlan.cancel")}
          </a>
        </div>
      </form>
    </div>
  );
}
