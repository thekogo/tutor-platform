import "server-only";
import createClient, { type Client } from "openapi-fetch";
import { cookies } from "next/headers";
import { forbidden, notFound, redirect } from "next/navigation";
import type { paths } from "./schema";
import { backendUrl, isApiMocking } from "@/lib/env";

/**
 * Server Component client: talks to BACKEND_URL directly, forwarding the
 * incoming request's cookies. In mock mode it dispatches to the MSW handlers
 * in-process instead (same handlers the browser hits via /api).
 */
export async function serverApi(): Promise<Client<paths>> {
  const cookieHeader = (await cookies()).toString();
  return createClient<paths>({
    baseUrl: isApiMocking ? "http://mock.local/api" : `${backendUrl()}/api`,
    headers: cookieHeader ? { cookie: cookieHeader } : undefined,
    cache: "no-store",
    fetch: isApiMocking
      ? async (req: Request) => (await import("@/mocks/dispatch")).handleMockRequest(req)
      : undefined,
  });
}

type FetchResult<T> = { data?: T; error?: unknown; response: Response };

/**
 * Returns data or interrupts rendering: 401 → /login, 403 → forbidden page,
 * 404 → not-found page, anything else → error boundary.
 */
export function unwrap<T>(result: FetchResult<T>): T {
  const { response } = result;
  if (response.ok && result.data !== undefined) return result.data;
  if (response.status === 401) redirect("/login");
  if (response.status === 403) forbidden();
  if (response.status === 404) notFound();
  throw new Error(`API ${response.url} → ${response.status}`);
}

/** Like unwrap, but 404 yields null (for "may not exist" resources). */
export function unwrapOptional<T>(result: FetchResult<T>): T | null {
  if (result.response.status === 404) return null;
  return unwrap(result);
}
