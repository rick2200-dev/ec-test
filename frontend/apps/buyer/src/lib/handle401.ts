"use client";

import { ApiError } from "@ec-marketplace/api-client";

/**
 * Inspect a thrown error and, when it's a 401 from the BFF (the gateway
 * proxy returns `code: "UNAUTHENTICATED"` when the Auth0 session is
 * missing / unrecoverable), redirect the browser to Auth0 login while
 * preserving the current path as `returnTo`.
 *
 * Returns true when the redirect was triggered so callers can short-
 * circuit their normal error UI; returns false otherwise so the caller
 * keeps responsibility for surfacing the error.
 *
 * Server components should use `redirect("/auth/login?returnTo=...")`
 * directly — this helper is client-only because it touches `window`.
 */
export function handle401(err: unknown): boolean {
  if (!(err instanceof ApiError)) return false;
  if (err.status !== 401) return false;
  if (typeof window === "undefined") return false;

  const returnTo = window.location.pathname + window.location.search;
  window.location.assign(`/auth/login?returnTo=${encodeURIComponent(returnTo)}`);
  return true;
}
