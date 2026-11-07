import { isApiMocking } from "@/lib/env";

/**
 * Mock mode only: serves /api/* from the MSW handlers in this Node process.
 * In normal mode src/proxy.ts rewrites /api/* to BACKEND_URL before this
 * route is reached.
 */
async function handle(request: Request): Promise<Response> {
  if (!isApiMocking) {
    return Response.json({ code: "bad_gateway", message: "API proxy is not configured" }, { status: 502 });
  }
  const { handleMockRequest } = await import("@/mocks/dispatch");
  return handleMockRequest(request);
}

export const dynamic = "force-dynamic";
export { handle as GET, handle as POST, handle as PUT, handle as PATCH, handle as DELETE };
