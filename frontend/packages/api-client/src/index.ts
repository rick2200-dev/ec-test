/**
 * Shared fetch primitives used by every frontend app to talk to the
 * gateway at `/api/v1/*`. App-specific endpoint wrappers (e.g.
 * `listBuyerInquiries`) stay in each app's `src/lib/api.ts` and
 * build on top of these.
 *
 * Why a shared module: all three apps (buyer, seller, admin) were
 * maintaining near-identical `fetchAPI` / error-handling code. A
 * single implementation means one place to add auth headers, retry
 * logic, or request tracing when we need them.
 */

/**
 * Base URL for gateway requests. Reads `NEXT_PUBLIC_API_URL` at build
 * time — Next.js inlines `NEXT_PUBLIC_*` variables when the app is
 * compiled, so this evaluates to a plain string in the final bundle.
 */
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

/**
 * ApiError carries the HTTP status alongside the parsed error body so
 * callers can branch on status (e.g. 403 → forbidden, 429 → rate
 * limited) without string-matching on the message. Always thrown from
 * `jsonOrThrow` on non-2xx responses.
 *
 * `code` holds the application-defined semantic error code when the
 * backend attached one (e.g. "DUPLICATE_EMAIL", "INVALID_OTP"). It lets
 * callers pick a display pattern for errors that share the same HTTP
 * status — two 400s that need different UI. Undefined when the server
 * did not set one, so legacy endpoints are unaffected.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly body: unknown;

  constructor(status: number, message: string, body: unknown = null, code?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.body = body;
  }
}

/**
 * Thin wrapper around `fetch` that prepends `API_BASE` and sets the
 * default JSON content type. Returns the raw `Response` so callers
 * can decide how to handle it (usually `jsonOrThrow`).
 */
export async function fetchAPI(path: string, options?: RequestInit): Promise<Response> {
  return fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
}

/**
 * Parses a JSON response, throwing `ApiError` on non-2xx. Understands
 * the gateway's `{error, code, message}` envelope and also handles 204s
 * by returning `undefined` (cast to `T` — callers using
 * `jsonOrThrow<void>` get the expected behavior).
 *
 * When the body carries a `code` field, it is promoted onto the thrown
 * `ApiError.code` so call sites can switch on it directly:
 *
 *     catch (e) {
 *       if (e instanceof ApiError && e.code === "DUPLICATE_EMAIL") ...
 *     }
 */
export async function jsonOrThrow<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let detail = "";
    let code: string | undefined;
    let parsed: unknown = null;
    try {
      parsed = await res.json();
      const body = parsed as {
        error?: string;
        message?: string;
        code?: string;
      };
      detail = body.error ?? body.message ?? "";
      code = body.code;
    } catch {
      // response wasn't JSON — fall back to status-based message
    }
    throw new ApiError(res.status, detail || `request failed: ${res.status}`, parsed, code);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}
