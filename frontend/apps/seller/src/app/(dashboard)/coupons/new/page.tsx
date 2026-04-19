"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import type { CouponDiscountType } from "@ec-marketplace/types";
import { createSellerCoupon } from "@/lib/api";

export default function NewSellerCouponPage() {
  const router = useRouter();
  const t = useTranslations("newCoupon");

  const [code, setCode] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [discountType, setDiscountType] = useState<CouponDiscountType>("percent");
  const [discountPercent, setDiscountPercent] = useState(10);
  const [discountAmount, setDiscountAmount] = useState(500);
  const currency = "JPY";
  const [minOrderAmount, setMinOrderAmount] = useState(0);
  const [maxDiscountAmount, setMaxDiscountAmount] = useState<number | "">("");
  const [usageLimitTotal, setUsageLimitTotal] = useState<number | "">("");
  const [usageLimitPerUser, setUsageLimitPerUser] = useState<number | "">("");
  const [expiresAt, setExpiresAt] = useState<string>("");

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const body = {
        code,
        title,
        description,
        discount_type: discountType,
        discount_percent_bps:
          discountType === "percent" ? Math.round(discountPercent * 100) : undefined,
        discount_amount: discountType === "fixed_amount" ? discountAmount : undefined,
        currency,
        min_order_amount: minOrderAmount,
        max_discount_amount:
          discountType === "percent" && typeof maxDiscountAmount === "number"
            ? maxDiscountAmount
            : undefined,
        usage_limit_total: typeof usageLimitTotal === "number" ? usageLimitTotal : undefined,
        usage_limit_per_user: typeof usageLimitPerUser === "number" ? usageLimitPerUser : undefined,
        expires_at_unix: expiresAt
          ? Math.floor(new Date(`${expiresAt}T23:59:59`).getTime() / 1000)
          : undefined,
      };
      await createSellerCoupon(body);
      router.push("/coupons");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || t("errorGeneric"));
      } else {
        setError(t("errorGeneric"));
      }
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-text-primary">{t("title")}</h2>
        <p className="mt-1 text-text-secondary">{t("description")}</p>
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
        className="space-y-6 rounded-lg border border-border bg-white p-6 shadow-sm"
      >
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("code")} <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={code}
              onChange={(e) => setCode(e.target.value.toUpperCase())}
              required
              className="w-full rounded-md border border-border px-3 py-2 font-mono text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("couponTitle")} <span className="text-danger">*</span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              required
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-text-primary">
            {t("couponDescription")}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <fieldset className="rounded-md border border-border p-4">
          <legend className="px-2 text-sm font-medium text-text-primary">
            {t("discount.legend")}
          </legend>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="discountType"
                value="percent"
                checked={discountType === "percent"}
                onChange={() => setDiscountType("percent")}
              />
              {t("discount.percent")}
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="discountType"
                value="fixed_amount"
                checked={discountType === "fixed_amount"}
                onChange={() => setDiscountType("fixed_amount")}
              />
              {t("discount.fixed")}
            </label>
          </div>

          {discountType === "percent" ? (
            <div className="mt-4 grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-text-primary">
                  {t("discount.percentLabel")}
                </label>
                <input
                  type="number"
                  min={0.01}
                  step={0.01}
                  value={discountPercent}
                  onChange={(e) => setDiscountPercent(Number(e.target.value))}
                  className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-text-primary">
                  {t("discount.maxAmount")}
                </label>
                <input
                  type="number"
                  min={0}
                  value={maxDiscountAmount}
                  onChange={(e) =>
                    setMaxDiscountAmount(e.target.value === "" ? "" : Number(e.target.value))
                  }
                  className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
                />
              </div>
            </div>
          ) : (
            <div className="mt-4">
              <label className="mb-1 block text-sm font-medium text-text-primary">
                {t("discount.fixedAmount")}
              </label>
              <input
                type="number"
                min={1}
                value={discountAmount}
                onChange={(e) => setDiscountAmount(Number(e.target.value))}
                className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              />
            </div>
          )}
        </fieldset>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("minOrderAmount")}
            </label>
            <input
              type="number"
              min={0}
              value={minOrderAmount}
              onChange={(e) => setMinOrderAmount(Number(e.target.value))}
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("expiresAt")}
            </label>
            <input
              type="date"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("usageLimitTotal")}
            </label>
            <input
              type="number"
              min={0}
              value={usageLimitTotal}
              onChange={(e) =>
                setUsageLimitTotal(e.target.value === "" ? "" : Number(e.target.value))
              }
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("usageLimitPerUser")}
            </label>
            <input
              type="number"
              min={0}
              value={usageLimitPerUser}
              onChange={(e) =>
                setUsageLimitPerUser(e.target.value === "" ? "" : Number(e.target.value))
              }
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
        </div>

        <div className="flex gap-3">
          <button
            type="submit"
            disabled={submitting}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-hover disabled:opacity-50"
          >
            {submitting ? t("creating") : t("create")}
          </button>
          <Link
            href="/coupons"
            className="rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-primary transition-colors hover:bg-surface-hover"
          >
            {t("cancel")}
          </Link>
        </div>
      </form>
    </div>
  );
}
