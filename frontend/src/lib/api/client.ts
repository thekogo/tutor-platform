import createClient, { type Client, type Middleware } from "openapi-fetch";
import type { paths } from "./schema";
import type { ApiError } from "./types";

/**
 * Browser-side typed client. Calls the same-origin `/api` (proxied to the Go
 * backend, or served by MSW in mock mode) so the session cookie stays first-party.
 */
const redirectOn401: Middleware = {
  onResponse({ response }) {
    if (response.status === 401 && typeof window !== "undefined") {
      const { pathname, search } = window.location;
      if (!pathname.startsWith("/login")) {
        // Hard navigation: works outside React (any fetch) and resets client state.
        // eslint-disable-next-line @next/next/no-location-assign-relative-destination
        window.location.assign(`/login?next=${encodeURIComponent(pathname + search)}`);
      }
    }
    return response;
  },
};

let client: Client<paths> | undefined;

export function api(): Client<paths> {
  if (!client) {
    const origin = typeof window === "undefined" ? "http://localhost:3000" : window.location.origin;
    client = createClient<paths>({ baseUrl: `${origin}/api`, credentials: "same-origin" });
    client.use(redirectOn401);
  }
  return client;
}

/** Best-effort Thai message for a failed API call. */
export function errorMessage(error: unknown, fallback = "เกิดข้อผิดพลาด กรุณาลองใหม่อีกครั้ง"): string {
  if (error && typeof error === "object" && "message" in error) {
    const msg = (error as ApiError).message;
    if (typeof msg === "string" && msg.trim()) return msg;
  }
  return fallback;
}
