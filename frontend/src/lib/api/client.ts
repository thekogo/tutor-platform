import createClient, { type Client, type Middleware } from "openapi-fetch";
import type { paths } from "./schema";
import type { ApiError } from "./types";

/**
 * Browser-side typed client. Calls the same-origin `/api` (proxied to the Go
 * backend by nginx in production, by the Next dev server in development) so the
 * session cookie stays first-party.
 */
const redirectOn401: Middleware = {
  onResponse({ response }) {
    if (response.status === 401 && typeof window !== "undefined") {
      const { pathname, search } = window.location;
      if (!pathname.startsWith("/login")) {
        // Hard navigation: works outside React (any fetch) and resets client state.
        const next = pathname === "/" ? "" : `?next=${encodeURIComponent(pathname + search)}`;
        // eslint-disable-next-line @next/next/no-location-assign-relative-destination
        window.location.assign(`/login${next}`);
      }
    }
    return response;
  },
};

let client: Client<paths> | undefined;

export function api(): Client<paths> {
  if (!client) {
    const origin = typeof window === "undefined" ? "http://localhost:3000" : window.location.origin;
    client = createClient<paths>({ baseUrl: `${origin}/api`, credentials: "same-origin", cache: "no-store" });
    client.use(redirectOn401);
  }
  return client;
}

/** A non-2xx API response, thrown by unwrap() and rendered by <Loaded>. */
export class HttpError extends Error {
  constructor(
    readonly status: number,
    url: string,
  ) {
    super(`API ${url} → ${status}`);
    this.name = "HttpError";
  }
}

type FetchResult<T> = { data?: T; error?: unknown; response: Response };

/** Returns data or throws HttpError (401 → the client already redirects to /login). */
export function unwrap<T>(result: FetchResult<T>): T {
  const { response } = result;
  if (response.ok && result.data !== undefined) return result.data;
  throw new HttpError(response.status, response.url);
}

/** Like unwrap, but 404 yields null (for "may not exist" resources). */
export function unwrapOptional<T>(result: FetchResult<T>): T | null {
  if (result.response.status === 404) return null;
  return unwrap(result);
}

const INVALIDATE_EVENT = "tp:invalidate";

/** Tells every mounted useLoad() to refetch (after a mutation). */
export function invalidate(): void {
  window.dispatchEvent(new Event(INVALIDATE_EVENT));
}

export function onInvalidate(listener: () => void): () => void {
  window.addEventListener(INVALIDATE_EVENT, listener);
  return () => window.removeEventListener(INVALIDATE_EVENT, listener);
}

/** Best-effort Thai message for a failed API call. */
export function errorMessage(error: unknown, fallback = "เกิดข้อผิดพลาด กรุณาลองใหม่อีกครั้ง"): string {
  if (error && typeof error === "object" && "message" in error) {
    const msg = (error as ApiError).message;
    if (typeof msg === "string" && msg.trim()) return msg;
  }
  return fallback;
}
