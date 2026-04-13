import type { NextRequest } from "next/server";
import { auth0 } from "./lib/auth0";

/**
 * The Auth0 SDK's middleware mounts /auth/login, /auth/callback,
 * /auth/logout, /auth/profile, and /auth/access-token on the app. It
 * also refreshes the session cookie (rolling expiry). Everything else
 * is passed through untouched.
 *
 * This file is the ONLY integration point for Auth0 routing — no
 * explicit /auth/* route files are needed.
 */
export async function middleware(request: NextRequest) {
  return auth0.middleware(request);
}

export const config = {
  matcher: [
    /*
     * Match every path EXCEPT:
     * - Next.js internals (_next/*)
     * - Static files (favicon.ico, robots.txt, images/fonts by extension)
     *
     * Without this the middleware would also fire on /_next/static/...
     * which is pointless and adds cookie-decrypt overhead to every
     * asset request.
     */
    "/((?!_next/static|_next/image|favicon.ico|robots.txt|sitemap.xml|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico|css|js|map)).*)",
  ],
};
