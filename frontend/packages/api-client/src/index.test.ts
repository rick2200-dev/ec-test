import { describe, it, expect, vi, beforeEach } from "vitest";
import { ApiError, fetchAPI, jsonOrThrow, API_BASE } from "./index";

describe("ApiError", () => {
  it("creates error with status and message", () => {
    const err = new ApiError(404, "not found");
    expect(err.status).toBe(404);
    expect(err.message).toBe("not found");
    expect(err.name).toBe("ApiError");
    expect(err.body).toBeNull();
    expect(err.code).toBeUndefined();
  });

  it("creates error with body and code", () => {
    const body = { error: "duplicate" };
    const err = new ApiError(409, "conflict", body, "DUPLICATE_EMAIL");
    expect(err.status).toBe(409);
    expect(err.body).toEqual(body);
    expect(err.code).toBe("DUPLICATE_EMAIL");
  });

  it("is an instance of Error", () => {
    const err = new ApiError(500, "server error");
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(ApiError);
  });
});

describe("fetchAPI", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("prepends API_BASE to path", async () => {
    const mockFetch = vi.fn().mockResolvedValue(new Response("ok"));
    vi.stubGlobal("fetch", mockFetch);

    await fetchAPI("/products");

    expect(mockFetch).toHaveBeenCalledTimes(1);
    const [url] = mockFetch.mock.calls[0];
    expect(url).toBe(`${API_BASE}/products`);
  });

  it("sets Content-Type to application/json by default", async () => {
    const mockFetch = vi.fn().mockResolvedValue(new Response("ok"));
    vi.stubGlobal("fetch", mockFetch);

    await fetchAPI("/test");

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers["Content-Type"]).toBe("application/json");
  });

  it("allows overriding headers", async () => {
    const mockFetch = vi.fn().mockResolvedValue(new Response("ok"));
    vi.stubGlobal("fetch", mockFetch);

    await fetchAPI("/test", {
      headers: { Authorization: "Bearer token123" },
    });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.headers["Authorization"]).toBe("Bearer token123");
    expect(options.headers["Content-Type"]).toBe("application/json");
  });

  it("passes through request options", async () => {
    const mockFetch = vi.fn().mockResolvedValue(new Response("ok"));
    vi.stubGlobal("fetch", mockFetch);

    await fetchAPI("/test", { method: "POST", body: JSON.stringify({ a: 1 }) });

    const [, options] = mockFetch.mock.calls[0];
    expect(options.method).toBe("POST");
    expect(options.body).toBe('{"a":1}');
  });
});

describe("jsonOrThrow", () => {
  it("parses JSON from successful response", async () => {
    const data = { id: 1, name: "Product" };
    const res = new Response(JSON.stringify(data), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });

    const result = await jsonOrThrow<typeof data>(res);
    expect(result).toEqual(data);
  });

  it("returns undefined for 204 No Content", async () => {
    const res = new Response(null, { status: 204 });
    const result = await jsonOrThrow<void>(res);
    expect(result).toBeUndefined();
  });

  it("throws ApiError on non-2xx with JSON error body", async () => {
    const body = { error: "not found", code: "ITEM_NOT_FOUND" };
    const res = new Response(JSON.stringify(body), {
      status: 404,
      headers: { "Content-Type": "application/json" },
    });

    try {
      await jsonOrThrow(res);
      expect.fail("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      const err = e as ApiError;
      expect(err.status).toBe(404);
      expect(err.message).toBe("not found");
      expect(err.code).toBe("ITEM_NOT_FOUND");
      expect(err.body).toEqual(body);
    }
  });

  it("throws ApiError with message field fallback", async () => {
    const body = { message: "validation failed" };
    const res = new Response(JSON.stringify(body), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });

    try {
      await jsonOrThrow(res);
      expect.fail("should have thrown");
    } catch (e) {
      const err = e as ApiError;
      expect(err.message).toBe("validation failed");
    }
  });

  it("throws ApiError with status-based message when body is not JSON", async () => {
    const res = new Response("Internal Server Error", {
      status: 500,
    });

    try {
      await jsonOrThrow(res);
      expect.fail("should have thrown");
    } catch (e) {
      const err = e as ApiError;
      expect(err.status).toBe(500);
      expect(err.message).toBe("request failed: 500");
    }
  });

  it("throws ApiError with no code when server omits code", async () => {
    const body = { error: "bad request" };
    const res = new Response(JSON.stringify(body), {
      status: 400,
      headers: { "Content-Type": "application/json" },
    });

    try {
      await jsonOrThrow(res);
      expect.fail("should have thrown");
    } catch (e) {
      const err = e as ApiError;
      expect(err.code).toBeUndefined();
    }
  });

  it("throws ApiError for 403 forbidden", async () => {
    const body = { error: "forbidden" };
    const res = new Response(JSON.stringify(body), {
      status: 403,
      headers: { "Content-Type": "application/json" },
    });

    try {
      await jsonOrThrow(res);
      expect.fail("should have thrown");
    } catch (e) {
      const err = e as ApiError;
      expect(err.status).toBe(403);
      expect(err.message).toBe("forbidden");
    }
  });
});
