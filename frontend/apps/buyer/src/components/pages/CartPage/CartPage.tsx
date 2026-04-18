"use client";

import { useCallback, useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError, clearCart, getCart, removeCartItem, updateCartItem } from "@/lib/api";
import type { CartItem } from "@ec-marketplace/types";
import { CartPagePresenter } from "./CartPage.presenter";

export default function CartPage() {
  const t = useTranslations("cart");
  const [items, setItems] = useState<CartItem[] | null>(null);
  const [needsLogin, setNeedsLogin] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingSku, setPendingSku] = useState<string | null>(null);
  const [clearing, setClearing] = useState(false);

  const handleApiError = useCallback(
    (err: unknown) => {
      if (err instanceof ApiError && err.status === 401) {
        setNeedsLogin(true);
        return;
      }
      setError(err instanceof Error ? err.message : t("errorGeneric"));
    },
    [t]
  );

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const cart = await getCart();
        if (cancelled) return;
        setItems(cart.items ?? []);
      } catch (err) {
        if (cancelled) return;
        handleApiError(err);
        // Render empty state rather than the loading shimmer so the user
        // can recover without a refresh.
        setItems([]);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [handleApiError]);

  const onChangeQuantity = async (skuId: string, nextQty: number) => {
    if (nextQty < 1 || pendingSku) return;
    setPendingSku(skuId);
    setError(null);
    try {
      const cart = await updateCartItem(skuId, nextQty);
      setItems(cart.items ?? []);
    } catch (err) {
      handleApiError(err);
    } finally {
      setPendingSku(null);
    }
  };

  const onRemove = async (skuId: string) => {
    if (pendingSku) return;
    setPendingSku(skuId);
    setError(null);
    try {
      const cart = await removeCartItem(skuId);
      setItems(cart.items ?? []);
    } catch (err) {
      handleApiError(err);
    } finally {
      setPendingSku(null);
    }
  };

  const onClear = async () => {
    if (clearing || !items || items.length === 0) return;
    if (typeof window !== "undefined" && !window.confirm(t("clearConfirm"))) {
      return;
    }
    setClearing(true);
    setError(null);
    try {
      const cart = await clearCart();
      setItems(cart.items ?? []);
    } catch (err) {
      handleApiError(err);
    } finally {
      setClearing(false);
    }
  };

  return (
    <CartPagePresenter
      title={t("title")}
      emptyTitle={t("emptyTitle")}
      emptyDescription={t("emptyDescription")}
      browseCta={t("browseCta")}
      itemColumnLabel={t("columns.item")}
      qtyColumnLabel={t("columns.quantity")}
      unitPriceColumnLabel={t("columns.unitPrice")}
      lineTotalColumnLabel={t("columns.lineTotal")}
      totalLabel={t("totalLabel")}
      checkoutCta={t("checkoutCta")}
      loadingLabel={t("loading")}
      loginRequiredTitle={t("loginRequiredTitle")}
      loginRequiredCta={t("loginRequiredCta")}
      items={items}
      needsLogin={needsLogin}
      errorSlot={
        error ? (
          <div
            role="alert"
            className="mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
          >
            {error}
          </div>
        ) : undefined
      }
      renderLineActions={(item) => (
        <div className="flex items-center justify-end gap-2">
          <button
            type="button"
            aria-label={t("actions.decrement")}
            disabled={item.quantity <= 1 || pendingSku === item.sku_id}
            onClick={() => onChangeQuantity(item.sku_id, item.quantity - 1)}
            className="h-7 w-7 rounded border border-gray-300 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-40"
          >
            −
          </button>
          <button
            type="button"
            aria-label={t("actions.increment")}
            disabled={pendingSku === item.sku_id}
            onClick={() => onChangeQuantity(item.sku_id, item.quantity + 1)}
            className="h-7 w-7 rounded border border-gray-300 text-sm text-gray-700 hover:bg-gray-50 disabled:opacity-40"
          >
            +
          </button>
          <button
            type="button"
            disabled={pendingSku === item.sku_id}
            onClick={() => onRemove(item.sku_id)}
            className="ml-2 text-xs font-medium text-red-600 hover:text-red-700 disabled:opacity-40"
          >
            {t("actions.remove")}
          </button>
        </div>
      )}
      footerActionsSlot={
        items && items.length > 0 ? (
          <button
            type="button"
            disabled={clearing}
            onClick={onClear}
            className="text-sm text-gray-600 hover:text-gray-900 disabled:opacity-40"
          >
            {clearing ? t("actions.clearing") : t("actions.clearAll")}
          </button>
        ) : undefined
      }
    />
  );
}
