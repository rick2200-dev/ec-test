"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError, addCartItem } from "@/lib/api";
import { AddToCartButtonPresenter } from "./AddToCartButton.presenter";

export interface AddToCartButtonProps {
  /** SKU UUID to add. When undefined the button is disabled. */
  skuId: string | undefined;
  /** Quantity to add per click. Defaults to 1. */
  quantity?: number;
  /** Notified after a successful add so the parent can refresh badges. */
  onAdded?: () => void;
}

const SUCCESS_FLASH_MS = 1500;

export default function AddToCartButton({ skuId, quantity = 1, onAdded }: AddToCartButtonProps) {
  const tProduct = useTranslations("product");
  const tAuth = useTranslations("auth");
  const tCart = useTranslations("cart");
  const [loading, setLoading] = useState(false);
  const [succeeded, setSucceeded] = useState(false);
  const [needsLogin, setNeedsLogin] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Reset the success flash after a short delay so the label flips back.
  useEffect(() => {
    if (!succeeded) return;
    const handle = window.setTimeout(() => setSucceeded(false), SUCCESS_FLASH_MS);
    return () => window.clearTimeout(handle);
  }, [succeeded]);

  const handleClick = async () => {
    if (!skuId || loading) return;
    setLoading(true);
    setError(null);
    try {
      await addCartItem(skuId, quantity);
      setSucceeded(true);
      onAdded?.();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401) {
          setNeedsLogin(true);
        } else {
          setError(err.message || tCart("errorGeneric"));
        }
      } else {
        setError(tCart("errorGeneric"));
      }
    } finally {
      setLoading(false);
    }
  };

  const loginReturnTo =
    typeof window === "undefined" ? "/cart" : window.location.pathname + window.location.search;

  return (
    <AddToCartButtonPresenter
      label={tProduct("addToCart")}
      loadingLabel={tCart("addingToCart")}
      successLabel={tCart("addedToCart")}
      loginCta={tAuth("loginToContinue")}
      disabled={!skuId}
      loading={loading}
      succeeded={succeeded}
      needsLogin={needsLogin}
      loginHref={`/auth/login?returnTo=${encodeURIComponent(loginReturnTo)}`}
      onClick={handleClick}
      errorMessage={error ?? undefined}
    />
  );
}
