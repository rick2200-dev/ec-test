"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

export default function NewPlanPage() {
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [tier, setTier] = useState(0);
  const [priceAmount, setPriceAmount] = useState(0);
  const [searchBoost, setSearchBoost] = useState(1.0);
  const [maxProducts, setMaxProducts] = useState(10);
  const [featuredSlots, setFeaturedSlots] = useState(0);
  const [promotedResults, setPromotedResults] = useState(0);
  const [stripePriceId, setStripePriceId] = useState("");
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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // TODO: API call to POST /api/v1/admin/plans
    alert(t("newPlan.createdMessage"));
  };

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("newPlan.title")}</h2>
        <p className="text-text-secondary mt-1">{t("newPlan.description")}</p>
      </div>

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
            <label htmlFor="priceAmount" className="block text-sm font-medium text-text-primary mb-1">
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
            <label htmlFor="searchBoost" className="block text-sm font-medium text-text-primary mb-1">
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
            <label htmlFor="maxProducts" className="block text-sm font-medium text-text-primary mb-1">
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
            <label htmlFor="featuredSlots" className="block text-sm font-medium text-text-primary mb-1">
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
            <label htmlFor="promotedResults" className="block text-sm font-medium text-text-primary mb-1">
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
          <label htmlFor="stripePriceId" className="block text-sm font-medium text-text-primary mb-1">
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
            className="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent-hover transition-colors text-sm font-medium"
          >
            {t("newPlan.create")}
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
