"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { ApiError, getCart } from "@/lib/api";
import { CartBadgePresenter } from "./CartBadge.presenter";

/**
 * Header cart icon. Fetches the cart count once on mount; falls back to
 * 0 on any error (including 401 — the badge stays mute for guests rather
 * than redirecting them to login). The /cart page itself enforces auth.
 */
export default function CartBadge() {
  const t = useTranslations("a11y");
  const [count, setCount] = useState(0);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const cart = await getCart();
        if (cancelled) return;
        const total = (cart.items ?? []).reduce((sum, item) => sum + item.quantity, 0);
        setCount(total);
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.status === 401) return;
        // Network or other errors: leave count at 0 silently — the badge
        // is decorative and shouldn't surface noise in the header.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return <CartBadgePresenter count={count} ariaLabel={t("cart")} />;
}
