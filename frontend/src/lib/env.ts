/** Mock mode: the whole API is served by in-process MSW handlers. */
export const isApiMocking = process.env.NEXT_PUBLIC_API_MOCKING === "enabled";

/** Show the email dev-login form on /login. */
export const isDevLoginEnabled = process.env.NEXT_PUBLIC_DEV_LOGIN === "true";

/** Server-only: base URL of the Go backend (no trailing slash). */
export function backendUrl(): string {
  return (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/+$/, "");
}

/** Name of the opaque session cookie set by the backend. */
export const SESSION_COOKIE = "tp_session";
