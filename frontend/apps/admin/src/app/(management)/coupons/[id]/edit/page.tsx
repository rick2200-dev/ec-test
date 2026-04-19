"use client";

import { use, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import type { Coupon } from "@ec-marketplace/types";
import { getAdminCoupon, updateAdminCoupon } from "@/lib/api";

interface RouteProps {
  params: Promise<{ id: string }>;
}

// Convert an ISO timestamp from the coupon row into the
// YYYY-MM-DD format the <input type="date"> expects.
function toDateInputValue(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export default function EditAdminCouponPage({ params }: RouteProps) {
  const { id } = use(params);
  const router = useRouter();
  const t = useTranslations("editCoupon");

  const [loading, setLoading] = useState(true);
  const [coupon, setCoupon] = useState<Coupon | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [minOrderAmount, setMinOrderAmount] = useState(0);
  const [maxDiscountAmount, setMaxDiscountAmount] = useState<number | "">("");
  const [usageLimitTotal, setUsageLimitTotal] = useState<number | "">("");
  const [usageLimitPerUser, setUsageLimitPerUser] = useState<number | "">("");
  const [expiresAt, setExpiresAt] = useState("");

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const c = await getAdminCoupon(id);
        if (cancelled) return;
        setCoupon(c);
        setTitle(c.title);
        setDescription(c.description);
        setMinOrderAmount(c.min_order_amount);
        setMaxDiscountAmount(c.max_discount_amount ?? "");
        setUsageLimitTotal(c.usage_limit_total ?? "");
        setUsageLimitPerUser(c.usage_limit_per_user ?? "");
        setExpiresAt(toDateInputValue(c.expires_at));
      } catch (err) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (submitting || !coupon) return;
    setSubmitting(true);
    setError(null);
    try {
      await updateAdminCoupon(id, {
        title,
        description,
        min_order_amount: minOrderAmount,
        max_discount_amount:
          coupon.discount_type === "percent" && typeof maxDiscountAmount === "number"
            ? maxDiscountAmount
            : null,
        usage_limit_total: typeof usageLimitTotal === "number" ? usageLimitTotal : null,
        usage_limit_per_user: typeof usageLimitPerUser === "number" ? usageLimitPerUser : null,
        expires_at_unix: expiresAt
          ? Math.floor(new Date(`${expiresAt}T23:59:59`).getTime() / 1000)
          : null,
      });
      router.push(`/coupons/${id}`);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || t("errorGeneric"));
      } else {
        setError(t("errorGeneric"));
      }
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="max-w-2xl">
        <p className="text-text-secondary">{t("loading")}</p>
      </div>
    );
  }
  if (loadError || !coupon) {
    return (
      <div className="max-w-2xl space-y-3">
        <p className="text-danger">{loadError ?? t("errorGeneric")}</p>
        <Link href="/coupons" className="text-sm text-accent hover:text-accent-hover">
          &larr; {t("backToList")}
        </Link>
      </div>
    );
  }

  const isPercent = coupon.discount_type === "percent";

  return (
    <div className="max-w-2xl space-y-6">
      <div>
        <Link href={`/coupons/${id}`} className="text-sm text-accent hover:text-accent-hover">
          &larr; {t("backToDetail")}
        </Link>
        <h2 className="mt-2 text-2xl font-bold text-text-primary">{t("title")}</h2>
        <p className="mt-1 font-mono text-sm text-text-secondary">{coupon.code}</p>
        <p className="mt-1 text-xs text-text-secondary">{t("immutableNote")}</p>
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
        <div>
          <label className="mb-1 block text-sm font-medium text-text-primary">
            {t("fields.title")} <span className="text-danger">*</span>
          </label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-text-primary">
            {t("fields.description")}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("fields.minOrderAmount")}
            </label>
            <input
              type="number"
              min={0}
              value={minOrderAmount}
              onChange={(e) => setMinOrderAmount(Number(e.target.value))}
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          {isPercent && (
            <div>
              <label className="mb-1 block text-sm font-medium text-text-primary">
                {t("fields.maxDiscountAmount")}
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
              <p className="mt-1 text-xs text-text-secondary">{t("fields.clearHint")}</p>
            </div>
          )}
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("fields.expiresAt")}
            </label>
            <input
              type="date"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <p className="mt-1 text-xs text-text-secondary">{t("fields.clearHint")}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("fields.usageLimitTotal")}
            </label>
            <input
              type="number"
              min={1}
              value={usageLimitTotal}
              onChange={(e) =>
                setUsageLimitTotal(e.target.value === "" ? "" : Number(e.target.value))
              }
              className="w-full rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <p className="mt-1 text-xs text-text-secondary">{t("fields.clearHint")}</p>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-primary">
              {t("fields.usageLimitPerUser")}
            </label>
            <input
              type="number"
              min={1}
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
            {submitting ? t("saving") : t("save")}
          </button>
          <Link
            href={`/coupons/${id}`}
            className="rounded-lg border border-border bg-surface px-4 py-2 text-sm font-medium text-text-primary transition-colors hover:bg-surface-hover"
          >
            {t("cancel")}
          </Link>
        </div>
      </form>
    </div>
  );
}
