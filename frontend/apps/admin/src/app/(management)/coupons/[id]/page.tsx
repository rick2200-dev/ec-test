import Link from "next/link";
import { getTranslations } from "next-intl/server";
import { notFound } from "next/navigation";
import { ApiError } from "@ec-marketplace/api-client";
import { getAdminCoupon, getAdminCouponStats } from "@/lib/api";
import type { Coupon, CouponStats } from "@ec-marketplace/types";
import RevokeCouponButton from "@/components/RevokeCouponButton";

interface RouteProps {
  params: Promise<{ id: string }>;
}

export const dynamic = "force-dynamic";

async function safeStats(id: string): Promise<CouponStats | null> {
  try {
    return await getAdminCouponStats(id);
  } catch {
    return null;
  }
}

export default async function CouponDetailPage({ params }: RouteProps) {
  const { id } = await params;
  const t = await getTranslations("couponDetail");

  let coupon: Coupon;
  try {
    coupon = await getAdminCoupon(id);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) notFound();
    throw err;
  }
  const stats = await safeStats(id);

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <Link href="/coupons" className="text-sm text-accent hover:text-accent-hover">
          &larr; {t("backToList")}
        </Link>
        <h2 className="mt-2 text-2xl font-bold text-text-primary">{coupon.title}</h2>
        <p className="mt-1 font-mono text-sm text-text-secondary">{coupon.code}</p>
      </div>

      <section className="rounded-lg border border-border bg-white p-6 shadow-sm">
        <h3 className="text-base font-semibold text-text-primary">{t("overview")}</h3>
        <dl className="mt-4 grid grid-cols-2 gap-4 text-sm">
          <div>
            <dt className="text-text-secondary">{t("status")}</dt>
            <dd
              className={`mt-1 inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                coupon.status === "active"
                  ? "bg-green-100 text-green-800"
                  : "bg-gray-200 text-gray-700"
              }`}
            >
              {coupon.status === "active" ? t("active") : t("revoked")}
            </dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("currency")}</dt>
            <dd className="mt-1 text-text-primary">{coupon.currency}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("discountType")}</dt>
            <dd className="mt-1 text-text-primary">
              {coupon.discount_type === "percent"
                ? `${((coupon.discount_percent_bps ?? 0) / 100).toFixed(2)}%`
                : `¥${(coupon.discount_amount ?? 0).toLocaleString()}`}
            </dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("minOrderAmount")}</dt>
            <dd className="mt-1 text-text-primary">¥{coupon.min_order_amount.toLocaleString()}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("validFrom")}</dt>
            <dd className="mt-1 text-text-primary">
              {new Date(coupon.valid_from).toLocaleString()}
            </dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("expiresAt")}</dt>
            <dd className="mt-1 text-text-primary">
              {coupon.expires_at ? new Date(coupon.expires_at).toLocaleString() : t("noExpiry")}
            </dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("usageLimitTotal")}</dt>
            <dd className="mt-1 text-text-primary">{coupon.usage_limit_total ?? t("unlimited")}</dd>
          </div>
          <div>
            <dt className="text-text-secondary">{t("usageLimitPerUser")}</dt>
            <dd className="mt-1 text-text-primary">
              {coupon.usage_limit_per_user ?? t("unlimited")}
            </dd>
          </div>
        </dl>
        {coupon.description && (
          <div className="mt-4 border-t border-border pt-4 text-sm text-text-primary">
            <p>{coupon.description}</p>
          </div>
        )}
      </section>

      <section className="rounded-lg border border-border bg-white p-6 shadow-sm">
        <h3 className="text-base font-semibold text-text-primary">{t("stats")}</h3>
        {stats ? (
          <dl className="mt-4 grid grid-cols-3 gap-4 text-sm">
            <div>
              <dt className="text-text-secondary">{t("redeemed")}</dt>
              <dd className="mt-1 text-2xl font-bold text-text-primary">{stats.redeemed_count}</dd>
            </div>
            <div>
              <dt className="text-text-secondary">{t("totalDiscount")}</dt>
              <dd className="mt-1 text-2xl font-bold text-text-primary">
                ¥{stats.total_discount_applied.toLocaleString()}
              </dd>
            </div>
            <div>
              <dt className="text-text-secondary">{t("pendingReservation")}</dt>
              <dd className="mt-1 text-2xl font-bold text-text-primary">
                {stats.pending_reservation_count}
              </dd>
            </div>
          </dl>
        ) : (
          <p className="mt-2 text-sm text-text-secondary">{t("statsUnavailable")}</p>
        )}
      </section>

      {coupon.status === "active" && <RevokeCouponButton id={coupon.id} />}
    </div>
  );
}
