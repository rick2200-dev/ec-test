"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { ApiError, checkoutCart, getCart, getPointsBalance, previewCoupon } from "@/lib/api";
import type {
  Cart,
  CouponPreviewResult,
  PointsBalance,
  SellerSubtotal,
} from "@ec-marketplace/types";
import {
  ShippingAddressFormPresenter,
  type ShippingAddressFields,
} from "./ShippingAddressForm.presenter";
import { CouponPreviewPanelPresenter } from "./CouponPreviewPanel.presenter";
import { PointsRedeemSliderPresenter } from "./PointsRedeemSlider.presenter";
import { OrderSummaryPresenter } from "./OrderSummary.presenter";

const EMPTY_ADDRESS: ShippingAddressFields = {
  recipient: "",
  postal_code: "",
  region: "",
  city: "",
  line1: "",
  line2: "",
  phone: "",
};

/** Sum quantity * unit_price grouped by seller_id. */
function buildSellerSubtotals(cart: Cart): SellerSubtotal[] {
  const bySeller = new Map<string, number>();
  for (const item of cart.items) {
    const prev = bySeller.get(item.seller_id) ?? 0;
    bySeller.set(item.seller_id, prev + item.unit_price_snapshot * item.quantity);
  }
  return Array.from(bySeller.entries()).map(([seller_id, subtotal]) => ({
    seller_id,
    subtotal,
  }));
}

export default function CheckoutPage() {
  const t = useTranslations("checkout");
  const tCart = useTranslations("cart");
  const router = useRouter();

  const [cart, setCart] = useState<Cart | null>(null);
  const [needsLogin, setNeedsLogin] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [address, setAddress] = useState<ShippingAddressFields>(EMPTY_ADDRESS);

  const [couponCode, setCouponCode] = useState("");
  const [coupon, setCoupon] = useState<CouponPreviewResult | null>(null);
  const [couponLoading, setCouponLoading] = useState(false);
  const [couponError, setCouponError] = useState<string | null>(null);

  const [pointsBalance, setPointsBalance] = useState<PointsBalance | null>(null);
  const [pointsToRedeem, setPointsToRedeem] = useState(0);

  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Initial cart + balance fetch.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const c = await getCart();
        if (cancelled) return;
        setCart(c);
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) {
          setNeedsLogin(true);
          return;
        }
        setLoadError(err instanceof Error ? err.message : tCart("errorGeneric"));
      }
      try {
        const b = await getPointsBalance();
        if (cancelled) return;
        setPointsBalance(b);
      } catch {
        // Balance is best-effort; failing here just means the slider
        // shows the no-points state.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [tCart]);

  const currency = cart?.items[0]?.currency ?? "JPY";
  const subtotal = useMemo(
    () =>
      cart
        ? cart.items.reduce((sum, item) => sum + item.unit_price_snapshot * item.quantity, 0)
        : 0,
    [cart]
  );
  const couponDiscount = coupon?.discount_amount ?? 0;
  const maxRedeemable = Math.max(0, subtotal - couponDiscount);
  const pointsActuallyUsed = Math.min(pointsToRedeem, maxRedeemable);
  const total = Math.max(0, subtotal - couponDiscount - pointsActuallyUsed);

  // Recompute coupon when cart changes after a successful preview.
  // We don't auto-revalidate on every line edit (that's the cart page's
  // job); the next "Apply" click re-runs against the latest snapshot.
  useEffect(() => {
    setPointsToRedeem((current) => Math.min(current, maxRedeemable));
  }, [maxRedeemable]);

  const handleApplyCoupon = async () => {
    if (!cart) return;
    const code = couponCode.trim();
    if (!code) return;
    setCouponLoading(true);
    setCouponError(null);
    try {
      const result = await previewCoupon({
        code,
        currency,
        seller_subtotals: buildSellerSubtotals(cart),
      });
      setCoupon(result);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "COUPON_NOT_FOUND" || err.status === 404) {
          setCouponError(t("coupon.errorNotFound"));
        } else if (err.code === "COUPON_MIN_ORDER") {
          setCouponError(t("coupon.errorMinOrder"));
        } else if (err.code === "COUPON_EXPIRED") {
          setCouponError(t("coupon.errorExpired"));
        } else if (err.code === "COUPON_USAGE_LIMIT") {
          setCouponError(t("coupon.errorUsageLimit"));
        } else if (err.status === 401) {
          setNeedsLogin(true);
        } else {
          setCouponError(err.message || t("coupon.errorGeneric"));
        }
      } else {
        setCouponError(t("coupon.errorGeneric"));
      }
    } finally {
      setCouponLoading(false);
    }
  };

  const handleClearCoupon = () => {
    setCoupon(null);
    setCouponError(null);
  };

  const handleAddressChange = (field: keyof ShippingAddressFields, value: string) =>
    setAddress((prev) => ({ ...prev, [field]: value }));

  const validateAddress = (): boolean => {
    const required: (keyof ShippingAddressFields)[] = [
      "recipient",
      "postal_code",
      "region",
      "city",
      "line1",
    ];
    return required.every((f) => address[f].trim().length > 0);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (submitting || !cart || cart.items.length === 0) return;
    if (!validateAddress()) {
      setSubmitError(t("errors.addressIncomplete"));
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      const result = await checkoutCart({
        shipping_address: { ...address },
        currency,
        ...(coupon ? { coupon_code: couponCode.trim() } : {}),
        ...(pointsActuallyUsed > 0 ? { points_to_redeem: pointsActuallyUsed } : {}),
      });
      const firstOrderId = result.order_ids[0] ?? "";
      router.push(`/checkout/complete?orderId=${encodeURIComponent(firstOrderId)}`);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401) {
          setNeedsLogin(true);
        } else if (err.code === "STOCK_INSUFFICIENT") {
          setSubmitError(t("errors.stockInsufficient"));
        } else if (err.code === "PRICE_MISMATCH") {
          setSubmitError(t("errors.priceMismatch"));
        } else {
          setSubmitError(err.message || t("errors.generic"));
        }
      } else {
        setSubmitError(t("errors.generic"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  if (needsLogin) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-12 text-center">
        <h1 className="text-2xl font-bold text-gray-900">{t("loginRequired.title")}</h1>
        <a
          href="/auth/login?returnTo=%2Fcheckout"
          className="mt-6 inline-block rounded-md bg-blue-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-blue-700"
        >
          {t("loginRequired.cta")}
        </a>
      </div>
    );
  }

  if (cart === null) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-12 text-center text-sm text-gray-500">
        {t("loading")}
      </div>
    );
  }

  if (cart.items.length === 0) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-12 text-center">
        <h1 className="text-2xl font-bold text-gray-900">{t("emptyCart.title")}</h1>
        <p className="mt-2 text-sm text-gray-500">{t("emptyCart.description")}</p>
        <Link
          href="/products"
          className="mt-6 inline-block rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
        >
          {t("emptyCart.browseCta")}
        </Link>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="text-2xl font-bold text-gray-900">{t("title")}</h1>
      {loadError && (
        <div
          className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          role="alert"
        >
          {loadError}
        </div>
      )}

      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-[2fr_1fr]">
        <div className="space-y-6">
          <ShippingAddressFormPresenter
            values={address}
            disabled={submitting}
            onChange={handleAddressChange}
            labels={{
              sectionTitle: t("shipping.title"),
              recipient: t("shipping.recipient"),
              postalCode: t("shipping.postalCode"),
              region: t("shipping.region"),
              city: t("shipping.city"),
              line1: t("shipping.line1"),
              line2: t("shipping.line2"),
              phone: t("shipping.phone"),
            }}
          />

          <CouponPreviewPanelPresenter
            code={couponCode}
            onCodeChange={setCouponCode}
            onApply={handleApplyCoupon}
            onClear={handleClearCoupon}
            loading={couponLoading}
            preview={coupon}
            error={couponError ?? undefined}
            labels={{
              title: t("coupon.title"),
              placeholder: t("coupon.placeholder"),
              apply: t("coupon.apply"),
              applying: t("coupon.applying"),
              clear: t("coupon.clear"),
              appliedDiscount: t("coupon.appliedDiscount"),
            }}
          />

          <PointsRedeemSliderPresenter
            spendable={pointsBalance?.spendable ?? 0}
            maxRedeemable={maxRedeemable}
            value={pointsToRedeem}
            onChange={setPointsToRedeem}
            labels={{
              title: t("points.title"),
              spendableLabel: t("points.spendable"),
              useLabel: t("points.use"),
              noPoints: t("points.none"),
            }}
          />
        </div>

        <aside className="space-y-4">
          <OrderSummaryPresenter
            itemsCount={cart.items.reduce((sum, i) => sum + i.quantity, 0)}
            subtotal={subtotal}
            couponDiscount={couponDiscount}
            pointsRedeemed={pointsActuallyUsed}
            total={total}
            labels={{
              title: t("summary.title"),
              items: t("summary.items"),
              subtotal: t("summary.subtotal"),
              couponDiscount: t("summary.couponDiscount"),
              pointsRedeemed: t("summary.pointsRedeemed"),
              total: t("summary.total"),
            }}
          />

          {submitError && (
            <div
              role="alert"
              className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
            >
              {submitError}
            </div>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-md bg-blue-600 px-5 py-3 text-base font-semibold text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {submitting ? t("submitting") : t("submit")}
          </button>
        </aside>
      </div>
    </form>
  );
}
