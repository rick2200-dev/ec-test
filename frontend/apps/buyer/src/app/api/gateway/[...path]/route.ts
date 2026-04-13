import { NextRequest, NextResponse } from "next/server";
import { auth0 } from "@/lib/auth0";

/**
 * BFF proxy for gateway calls. The browser talks to the Next.js server
 * on the same origin (/api/gateway/...), which lets us keep the Auth0
 * session cookie HttpOnly. This handler pulls the access_token out of
 * the session and forwards the request to the Go gateway with
 * `Authorization: Bearer <token>`.
 *
 * The gateway performs JWKS verification against Auth0 — no further
 * auth work is needed here beyond "is there a session at all". When
 * the user is logged out we respond 401 so the buyer app can route to
 * the login page instead of issuing a token-less upstream call that
 * would 401 anyway.
 */

const GATEWAY_URL = process.env.GATEWAY_URL ?? "http://localhost:8080";

// Headers we must never forward to the upstream — hop-by-hop headers
// (RFC 7230 §6.1) and anything the fetch implementation will regenerate.
// `host` is critical: leaking the Next.js Host through breaks the
// gateway's virtual-host routing if it ever matters.
const STRIP_REQUEST_HEADERS = new Set([
  "host",
  "connection",
  "content-length",
  "transfer-encoding",
  "keep-alive",
  "upgrade",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "cookie",
  "authorization",
]);

// Response headers that fetch already handles for us. Re-emitting
// `content-encoding` in particular causes browsers to double-decode.
const STRIP_RESPONSE_HEADERS = new Set([
  "connection",
  "content-encoding",
  "content-length",
  "transfer-encoding",
  "keep-alive",
]);

async function handle(request: NextRequest, params: { path: string[] }) {
  // Use getAccessToken() rather than reading session.tokenSet.accessToken
  // directly: the SDK transparently refreshes an expired access token
  // using the refresh token (offline_access scope) and persists the new
  // token set back into the session cookie. Reading the raw field would
  // otherwise 401 mid-session whenever Auth0's short-lived access
  // tokens (24h by default) expire before the session cookie.
  let accessToken: string;
  try {
    const token = await auth0.getAccessToken();
    accessToken = typeof token === "string" ? token : token.token;
  } catch {
    return NextResponse.json(
      { error: "not authenticated", code: "UNAUTHENTICATED" },
      { status: 401 }
    );
  }
  if (!accessToken) {
    return NextResponse.json(
      { error: "not authenticated", code: "UNAUTHENTICATED" },
      { status: 401 }
    );
  }

  const subPath = params.path?.join("/") ?? "";
  const url = new URL(request.url);
  const upstream = `${GATEWAY_URL}/${subPath}${url.search}`;

  const headers = new Headers();
  for (const [key, value] of request.headers.entries()) {
    if (!STRIP_REQUEST_HEADERS.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  }
  headers.set("Authorization", `Bearer ${accessToken}`);

  // Only read the body when the method actually has one. Reading a
  // body-less GET throws "TypeError: Failed to execute 'clone'" in
  // some runtimes.
  const method = request.method.toUpperCase();
  const hasBody = method !== "GET" && method !== "HEAD";

  const init: RequestInit = {
    method,
    headers,
    body: hasBody ? await request.arrayBuffer() : undefined,
    // Follow redirects manually so the browser can observe them.
    redirect: "manual",
  };

  const upstreamRes = await fetch(upstream, init);

  const respHeaders = new Headers();
  for (const [key, value] of upstreamRes.headers.entries()) {
    if (!STRIP_RESPONSE_HEADERS.has(key.toLowerCase())) {
      respHeaders.set(key, value);
    }
  }

  return new NextResponse(upstreamRes.body, {
    status: upstreamRes.status,
    statusText: upstreamRes.statusText,
    headers: respHeaders,
  });
}

// Next.js 16 wraps `params` in a Promise; await it once per method.
export async function GET(request: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return handle(request, await ctx.params);
}
export async function POST(request: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return handle(request, await ctx.params);
}
export async function PUT(request: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return handle(request, await ctx.params);
}
export async function PATCH(request: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return handle(request, await ctx.params);
}
export async function DELETE(request: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return handle(request, await ctx.params);
}
