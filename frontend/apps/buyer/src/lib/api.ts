const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function fetchAPI(path: string, options?: RequestInit) {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  return res;
}

export async function trackEvent(eventType: string, productId: string) {
  try {
    await fetchAPI("/api/v1/buyer/events", {
      method: "POST",
      body: JSON.stringify({ event_type: eventType, product_id: productId }),
    });
  } catch {
    // Silently fail - tracking should not break the UI
  }
}
