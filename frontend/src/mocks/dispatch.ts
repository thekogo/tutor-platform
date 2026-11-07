import { getResponse } from "msw";
import { handlers } from "./handlers";

/** Resolve a Request against the MSW handlers (mock mode, server side). */
export async function handleMockRequest(request: Request): Promise<Response> {
  const res = await getResponse(handlers, request);
  if (!res) {
    return Response.json({ code: "not_found", message: `No mock for ${request.method} ${new URL(request.url).pathname}` }, { status: 404 });
  }
  // Re-wrap so framework code gets a plain Response (incl. Set-Cookie / Location).
  return new Response(res.body, { status: res.status, statusText: res.statusText, headers: res.headers });
}
