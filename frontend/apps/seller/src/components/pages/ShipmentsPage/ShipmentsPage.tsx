"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { ApiError } from "@ec-marketplace/api-client";
import { listSellerShipments, markShipmentDelivered, registerShipment } from "@/lib/api";
import type { Shipment, ShipmentStatus } from "@ec-marketplace/types";
import { ShipmentsListPresenter } from "./ShipmentsList.presenter";
import { RegisterTrackingModalPresenter } from "./RegisterTrackingModal.presenter";

const STATUS_FILTERS: (ShipmentStatus | "all")[] = [
  "all",
  "pending",
  "ready_to_ship",
  "shipped",
  "delivered",
  "cancelled",
];

export default function ShipmentsPage() {
  const t = useTranslations("shipments");
  const locale = useLocale();

  const [statusFilter, setStatusFilter] = useState<ShipmentStatus | "all">("all");
  const [shipments, setShipments] = useState<Shipment[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Register-tracking modal state.
  const [registeringFor, setRegisteringFor] = useState<Shipment | null>(null);
  const [carrier, setCarrier] = useState("");
  const [trackingNumber, setTrackingNumber] = useState("");
  const [note, setNote] = useState("");
  const [registerSubmitting, setRegisterSubmitting] = useState(false);
  const [registerError, setRegisterError] = useState<string | null>(null);

  // Per-row "marking delivered" state — used to disable the button while
  // the call is in flight.
  const [deliveringId, setDeliveringId] = useState<string | null>(null);

  const reload = async () => {
    setLoaded(false);
    setError(null);
    try {
      const res = await listSellerShipments({ status: statusFilter, limit: 50 });
      setShipments(res.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.loadFailed"));
    } finally {
      setLoaded(true);
    }
  };

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await listSellerShipments({ status: statusFilter, limit: 50 });
        if (cancelled) return;
        setShipments(res.items ?? []);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : t("errors.loadFailed"));
      } finally {
        if (!cancelled) setLoaded(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [statusFilter, t]);

  const openRegisterModal = (s: Shipment) => {
    setRegisteringFor(s);
    setCarrier(s.carrier ?? "");
    setTrackingNumber(s.tracking_number ?? "");
    setNote(s.note ?? "");
    setRegisterError(null);
  };

  const closeRegisterModal = () => {
    if (registerSubmitting) return;
    setRegisteringFor(null);
  };

  const submitRegister = async () => {
    if (!registeringFor || registerSubmitting) return;
    setRegisterSubmitting(true);
    setRegisterError(null);
    try {
      await registerShipment(registeringFor.id, {
        carrier: carrier.trim(),
        tracking_number: trackingNumber.trim(),
        note: note.trim(),
      });
      setRegisteringFor(null);
      await reload();
    } catch (err) {
      if (err instanceof ApiError) {
        setRegisterError(err.message || t("errors.registerFailed"));
      } else {
        setRegisterError(t("errors.registerFailed"));
      }
    } finally {
      setRegisterSubmitting(false);
    }
  };

  const handleDeliver = async (s: Shipment) => {
    if (deliveringId) return;
    setDeliveringId(s.id);
    try {
      await markShipmentDelivered(s.id);
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("errors.deliverFailed"));
    } finally {
      setDeliveringId(null);
    }
  };

  const statusLabels: Record<ShipmentStatus, string> = {
    pending: t("status.pending"),
    ready_to_ship: t("status.readyToShip"),
    shipped: t("status.shipped"),
    delivered: t("status.delivered"),
    cancelled: t("status.cancelled"),
  };

  return (
    <div className="space-y-6">
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

      <div className="flex items-center gap-2">
        <label htmlFor="ship-filter" className="text-sm font-medium text-text-secondary">
          {t("filterLabel")}
        </label>
        <select
          id="ship-filter"
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as ShipmentStatus | "all")}
          className="rounded-md border border-border px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
        >
          {STATUS_FILTERS.map((s) => (
            <option key={s} value={s}>
              {s === "all" ? t("filter.all") : statusLabels[s]}
            </option>
          ))}
        </select>
      </div>

      <ShipmentsListPresenter
        shipments={shipments}
        emptyLabel={t("empty")}
        loadingLabel={t("loading")}
        loaded={loaded}
        statusLabels={statusLabels}
        columns={{
          orderId: t("columns.orderId"),
          status: t("columns.status"),
          carrier: t("columns.carrier"),
          tracking: t("columns.tracking"),
          shippedAt: t("columns.shippedAt"),
          actions: t("columns.actions"),
        }}
        renderActions={(s) => (
          <div className="flex items-center justify-end gap-2">
            {(s.status === "pending" || s.status === "ready_to_ship") && (
              <button
                type="button"
                onClick={() => openRegisterModal(s)}
                className="rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-hover"
              >
                {t("actions.register")}
              </button>
            )}
            {s.status === "shipped" && (
              <button
                type="button"
                onClick={() => handleDeliver(s)}
                disabled={deliveringId === s.id}
                className="rounded-md border border-accent px-3 py-1.5 text-xs font-medium text-accent hover:bg-surface-hover disabled:opacity-50"
              >
                {deliveringId === s.id ? t("actions.delivering") : t("actions.deliver")}
              </button>
            )}
          </div>
        )}
        locale={locale}
      />

      <RegisterTrackingModalPresenter
        open={registeringFor !== null}
        carrier={carrier}
        trackingNumber={trackingNumber}
        note={note}
        submitting={registerSubmitting}
        onCarrierChange={setCarrier}
        onTrackingNumberChange={setTrackingNumber}
        onNoteChange={setNote}
        onSubmit={submitRegister}
        onClose={closeRegisterModal}
        errorMessage={registerError ?? undefined}
        labels={{
          title: t("register.title"),
          carrier: t("register.carrier"),
          trackingNumber: t("register.trackingNumber"),
          note: t("register.note"),
          submit: t("register.submit"),
          submitting: t("register.submitting"),
          cancel: t("register.cancel"),
        }}
      />
    </div>
  );
}
