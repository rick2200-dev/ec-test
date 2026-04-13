import { Auth0Client } from "@auth0/nextjs-auth0/server";
import { NextResponse } from "next/server";

/**
 * Single Auth0 client shared by the BFF (middleware, callback, session
 * lookup inside /api/gateway). The SDK reads its configuration from env
 * vars (AUTH0_DOMAIN, AUTH0_CLIENT_ID, AUTH0_CLIENT_SECRET,
 * AUTH0_SECRET, APP_BASE_URL, plus AUTH0_AUDIENCE + AUTH0_SCOPE below).
 *
 * onCallback: fires after Auth0 exchanges the code for tokens and
 * before the user is redirected back into the app. We use it to upsert
 * the buyer profile in the auth service so the row exists before any
 * authenticated call lands. A failure here MUST NOT block login —
 * downstream services key off the Auth0 `sub` directly; the local row
 * is only a profile-display convenience.
 */
export const auth0 = new Auth0Client({
  authorizationParameters: {
    // Requesting `audience` causes Auth0 to issue an access_token signed
    // for our API (JWT) rather than a generic opaque token. Without it
    // the gateway's JWKS verification would reject the token.
    audience: process.env.AUTH0_AUDIENCE,
    scope: process.env.AUTH0_SCOPE ?? "openid profile email",
  },
  async onCallback(error, context, session) {
    const appBaseURL = process.env.APP_BASE_URL ?? "http://localhost:3000";
    const returnTo = context?.returnTo ?? "/";
    if (error) {
      // Bubble the error page up; don't attempt to upsert with no session.
      return NextResponse.redirect(
        new URL(
          `/auth/error?error=${encodeURIComponent(error.message ?? "auth_error")}`,
          appBaseURL
        )
      );
    }

    try {
      const sub = session?.user?.sub;
      const email = session?.user?.email;
      if (!sub || !email) {
        return NextResponse.redirect(new URL(returnTo, appBaseURL));
      }
      const tenantID = process.env.DEFAULT_TENANT_ID;
      const internalToken = process.env.AUTH_INTERNAL_TOKEN;
      const authURL = process.env.AUTH_SERVICE_URL;
      if (!tenantID || !internalToken || !authURL) {
        // Misconfigured — log and continue. Login still succeeds.
        console.warn("buyer upsert skipped: missing env", {
          hasTenant: Boolean(tenantID),
          hasToken: Boolean(internalToken),
          hasURL: Boolean(authURL),
        });
        return NextResponse.redirect(new URL(returnTo, appBaseURL));
      }

      const res = await fetch(`${authURL}/internal/buyers/upsert`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Internal-Token": internalToken,
        },
        body: JSON.stringify({
          tenant_id: tenantID,
          auth0_sub: sub,
          email,
          display_name: session?.user?.name ?? "",
        }),
      });
      if (!res.ok) {
        console.warn("buyer upsert failed", {
          status: res.status,
          body: await res.text().catch(() => ""),
        });
      }
    } catch (e) {
      // Never block login. The next call will retry on the user's next login.
      console.warn("buyer upsert error", e);
    }
    return NextResponse.redirect(new URL(returnTo, appBaseURL));
  },
});
