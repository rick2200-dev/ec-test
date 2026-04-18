"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import { revokeAdminCoupon } from "@/lib/api";

export default function RevokeCouponButton({ id }: { id: string }) {
  const t = useTranslations("couponDetail");
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onClick = async () => {
    if (pending) return;
    if (typeof window !== "undefined" && !window.confirm(t("confirmRevoke"))) return;
    setPending(true);
    setError(null);
    try {
      await revokeAdminCoupon(id);
      router.refresh();
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || t("revokeFailed"));
      } else {
        setError(t("revokeFailed"));
      }
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="rounded-lg border border-red-200 bg-red-50 p-4">
      <h3 className="text-sm font-semibold text-danger">{t("dangerZone")}</h3>
      <p className="mt-1 text-xs text-red-700">{t("revokeNotice")}</p>
      <button
        type="button"
        onClick={onClick}
        disabled={pending}
        className="mt-3 rounded-md bg-danger px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50"
      >
        {pending ? t("revoking") : t("revoke")}
      </button>
      {error && (
        <p role="alert" className="mt-2 text-xs text-danger">
          {error}
        </p>
      )}
    </div>
  );
}
